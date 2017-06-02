package flannel

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	microerror "github.com/giantswarm/microkit/error"
	micrologger "github.com/giantswarm/microkit/logger"
	"github.com/giantswarm/operatorkit/tpr"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	k8serrors "k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	"github.com/giantswarm/flannel-operator/flag"
	"github.com/giantswarm/flannel-operator/service/flannel/k8sutil"
	"github.com/giantswarm/flanneltpr"
)

// Config represents the configuration used to create a Crt service.
type Config struct {
	// Dependencies.
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// Settings.
	Flag  *flag.Flag
	Viper *viper.Viper
}

// DefaultConfig provides a default configuration to create a new create service
// by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		K8sClient: nil,
		Logger:    nil,

		// Settings.
		Flag:  nil,
		Viper: nil,
	}
}

// Service implements the Crt service interface.
type Service struct {
	Config

	// Internals.
	bootOnce sync.Once
	tpr      *tpr.TPR
}

// New creates a new configured Crt service.
func New(config Config) (*Service, error) {
	// Dependencies.
	if config.K8sClient == nil {
		return nil, microerror.MaskAnyf(invalidConfigError, "kubernetes client must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.MaskAnyf(invalidConfigError, "logger must not be empty")
	}

	// Settings.
	if config.Flag == nil {
		return nil, microerror.MaskAnyf(invalidConfigError, "flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.MaskAnyf(invalidConfigError, "viper must not be empty")
	}

	tpr, err := tpr.New(tpr.Config{
		Clientset: config.K8sClient,

		Name:        flanneltpr.Name,
		Version:     flanneltpr.VersionV1,
		Description: flanneltpr.Description,
	})
	if err != nil {
		return nil, microerror.MaskAnyf(err, "creating TPR util")
	}

	newService := &Service{
		Config: config,

		// Internals
		bootOnce: sync.Once{},
		tpr:      tpr,
	}

	return newService, nil
}

// Boot starts the service and implements the watch for the flannel TPR.
func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		err := s.tpr.CreateAndWait()
		switch {
		case tpr.IsAlreadyExists(err):
			s.Logger.Log("info", "flannel third-party resource already exists")
		case err != nil:
			panic(fmt.Sprintf("could not create flannel resource: %#v", err))
		default:
			s.Logger.Log("info", "successfully created flannel third-party resource")
		}
	})

	var informer *cache.Controller
	{
		zeroObject := k8sutil.ZeroObjectFactoryFuncs{
			NewObjectFunc:     func() runtime.Object { return new(flanneltpr.CustomObject) },
			NewObjectListFunc: func() runtime.Object { return new(flanneltpr.List) },
		}
		observer := k8sutil.ObserverFuncs{
			OnListFunc: func() {
				s.Logger.Log("debug", "executing the reconciler's list function", "event", "list")
			},
			OnWatchFunc: func() {
				s.Logger.Log("debug", "executing the reconciler's watch function", "event", "watch")
			},
		}
		handler := cache.ResourceEventHandlerFuncs{
			AddFunc:    s.addFunc,
			DeleteFunc: s.deleteFunc,
		}
		informer = k8sutil.NewInformer(s.K8sClient, s.tpr, zeroObject, observer, handler)
	}

	s.Logger.Log("debug", "starting list/watch")
	informer.Run(nil)
}

// addFunc does nothing as the operator reacts only on TPO delete.
func (s *Service) addFunc(obj interface{}) {}

// deleteFunc waits for the delting cluster's namespace to be fully deleted and
// then cleans up flannel bridges.
func (s *Service) deleteFunc(obj interface{}) {
	var spec flanneltpr.Spec
	{
		o, ok := obj.(*flanneltpr.CustomObject)
		if !ok {
			err := microerror.MaskAnyf(wrongTypeError, "expected '%T', got '%T'", &flanneltpr.CustomObject{}, obj)
			s.Logger.Log("error", fmt.Sprintf("%#v", err))
			return
		}
		spec = o.Spec
	}

	// Wait for the cluster's namespace to be deleted.
	{
		// op does not mask errors, they are used only to be logged in notify.
		op := func() error {
			_, err := s.K8sClient.CoreV1().Namespaces().Get(spec.Namespace)
			if err != nil && k8serrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			return errors.New("still exists")
		}

		notify := func(reason error, interval time.Duration) {
			s.Logger.Log("debug", "waiting for the namespace to be removed", "reason", reason.Error(), "namespace", spec.Namespace)
		}

		backoff.RetryNotify(op, backoff.NewExponentialBackOff(), notify)
	}

	s.Logger.Log("debug", "cluster namespace deleted, cleaning flannel resources", "namespace", spec.Namespace)

	// Schedule flannel resources cleanup on every node using deployment.
	var podAffinity string
	{
		pa := newPodAffinity(spec)
		data, err := json.Marshal(pa)
		if err != nil {
			s.Logger.Log("error", fmt.Sprintf("%#v", microerror.MaskAnyf(err, "marshalling podAffinity JSON")))
			return
		}
		podAffinity = string(data)
	}

	var replicas int32
	{
		nodes, err := s.K8sClient.CoreV1().Nodes().List(v1.ListOptions{})
		if err != nil {
			s.Logger.Log("error", fmt.Sprintf("%#v", microerror.MaskAnyf(err, "requesting cluster node list")))
			return
		}
		// Assuming master node is running a kubelet.
		replicas = int32(len(nodes.Items))
	}

	var deployment *v1beta1.Deployment
	{
		deployment = newDeployment(spec, replicas)
		deployment.Spec.Template.Annotations["scheduler.alpha.kubernetes.io/affinity"] = podAffinity
	}

	_, err := s.K8sClient.ExtensionsV1beta1().Deployments(namespace).Create(deployment)
	if err != nil {
		s.Logger.Log("error", fmt.Sprintf("%#v", microerror.MaskAnyf(err, "creating deployment %s", deployment.Name)))
		return
	}

	// Wait for the cleanup to complete and delete pods.
	{
		// op does not mask errors, they are used only to be logged in notify.
		op := func() error {
			opts := v1.ListOptions{
				LabelSelector: "app=" + deployment.Spec.Template.ObjectMeta.Labels["app"],
			}
			pods, err := s.K8sClient.CoreV1().Pods(namespace).List(opts)
			if err != nil {
				return microerror.MaskAnyf(err, "requesting cluster pod list")
			}
			succeeded := 0
			for _, p := range pods.Items {
				if p.Status.Phase == v1.PodSucceeded {
					succeeded++
				}
			}
			if succeeded != len(pods.Items) {
				return fmt.Errorf("flannel cleanup in progress %d/%d", succeeded, len(pods.Items))
			}
			return nil
		}

		notify := func(reason error, interval time.Duration) {
			s.Logger.Log("debug", "waiting for the namespace to be removed", "reason", reason.Error(), "namespace", spec.Namespace)
		}

		backoff.RetryNotify(op, backoff.NewExponentialBackOff(), notify)
	}

	// Cleanup pods.
	{
		opts := v1.ListOptions{
			LabelSelector: "app=" + deployment.Spec.Template.ObjectMeta.Labels["app"],
		}
		err := s.K8sClient.CoreV1().Pods(namespace).DeleteCollection(&v1.DeleteOptions{}, opts)
		if err != nil {
			s.Logger.Log("error", fmt.Sprintf("%#v", microerror.MaskAnyf(err, "deleting flannel cleanup pods")))
			return
		}
	}

	s.Logger.Log("info", "finished flannel cleanup for cluster %s", spec.Namespace)
}
