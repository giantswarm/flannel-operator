package flannel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	microerror "github.com/giantswarm/microkit/error"
	micrologger "github.com/giantswarm/microkit/logger"
	"github.com/giantswarm/microkit/storage"
	"github.com/giantswarm/microkit/tls"
	"github.com/giantswarm/operatorkit/tpr"
	"github.com/spf13/viper"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/giantswarm/flannel-operator/flag"
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

	var err error

	var newTPR *tpr.TPR
	{
		tprConfig := tpr.DefaultConfig()

		tprConfig.K8sClient = config.K8sClient
		tprConfig.Logger = config.Logger

		tprConfig.Description = flanneltpr.Description
		tprConfig.Name = flanneltpr.Name
		tprConfig.Version = flanneltpr.VersionV1

		newTPR, err = tpr.New(tprConfig)
		if err != nil {
			return nil, microerror.MaskAny(err)
		}
	}

	var store storage.Service
	{
		endpoint := config.Viper.GetString(config.Flag.Service.Etcd.Endpoint)
		rootCAs := []string{}
		{
			v := config.Viper.GetString(config.Flag.Service.Etcd.TLS.CAFile)
			if v != "" {
				rootCAs = []string{v}
			}
		}
		certFiles := tls.CertFiles{
			RootCAs: rootCAs,
			Cert:    config.Viper.GetString(config.Flag.Service.Etcd.TLS.CrtFile),
			Key:     config.Viper.GetString(config.Flag.Service.Etcd.TLS.KeyFile),
		}

		config := storage.DefaultConfig()
		config.EtcdAddress = endpoint
		config.EtcdTLS = certFiles
		config.Kind = storage.KindEtcdV2

		store, err = storage.New(config)
		if err != nil {
			return nil, microerror.MaskAnyf(err, "creating storage for etcd endpoint %s and certificates %#v", endpoint, certFiles)
		}
	}

	newService := &Service{
		Config: config,

		// Internals
		bootOnce: sync.Once{},
		tpr:      newTPR,
		store:    store,
	}

	return newService, nil
}

// Service implements the Flannel service.
type Service struct {
	Config

	// Internals.
	bootOnce sync.Once
	tpr      *tpr.TPR
	store    storage.Service
}

// Boot starts the service and implements the watch for the flannel TPR.
func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		err := s.tpr.CreateAndWait()
		if tpr.IsAlreadyExists(err) {
			s.Logger.Log("debug", "third party resource already exists")
		} else if err != nil {
			s.Logger.Log("error", fmt.Sprintf("%#v", err))
			return
		}

		s.Logger.Log("debug", "starting list/watch")

		newResourceEventHandler := &cache.ResourceEventHandlerFuncs{
			AddFunc:    s.addFunc,
			DeleteFunc: s.deleteFunc,
		}
		newZeroObjectFactory := &tpr.ZeroObjectFactoryFuncs{
			NewObjectFunc:     func() runtime.Object { return &flanneltpr.CustomObject{} },
			NewObjectListFunc: func() runtime.Object { return &flanneltpr.List{} },
		}

		s.tpr.NewInformer(newResourceEventHandler, newZeroObjectFactory).Run(nil)
	})
}

// addFunc does nothing as the operator reacts only on TPO delete.
func (s *Service) addFunc(obj interface{}) {
	defer func() {
		if r := recover(); r != nil {
			s.Logger.Log("error", "recovered in addFunc", "panic", r, "event", "add")
		}
	}()
	err := s.addFuncError(obj)
	if err != nil {
		s.Logger.Log("error", fmt.Sprintf("%#v", err), "event", "add")
	}
}

func (s *Service) addFuncError(obj interface{}) error {
	var spec flanneltpr.Spec
	{
		o, ok := obj.(*flanneltpr.CustomObject)
		if !ok {
			return microerror.MaskAnyf(wrongTypeError, "expected '%T', got '%T'", &flanneltpr.CustomObject{}, obj)
		}
		spec = o.Spec
	}

	// Create flannel etcd config.
	{
		path := etcdPath(spec)

		type flannelBackend struct {
			Type string
			VNI  int
		}

		type flannelConfig struct {
			Network   string
			SubnetLen int
			Backend   flannelBackend
		}

		config := flannelConfig{
			Network:   spec.Flannel.Config.Network,
			SubnetLen: spec.Flannel.Config.SubnetLen,
			Backend: flannelBackend{
				Type: "vxlan",
				VNI:  spec.Flannel.Config.VNI,
			},
		}

		bytes, err := json.Marshal(config)
		if err != nil {
			return microerror.MaskAnyf(err, "marshaling %#v", config)
		}

		exists, err := s.store.Exists(context.TODO(), path)
		if err != nil {
			return microerror.MaskAnyf(err, "checking %s etcd key existence", path)
		}
		if exists {
			s.Logger.Log("debug", "etcd key "+path+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else {
			err := s.store.Create(context.TODO(), path, string(bytes))
			if err != nil {
				return microerror.MaskAnyf(err, "createing %s etcd key", path)
			}
		}
	}

	// Create namespace for the cleanup job.
	{
		ns := newNamespace(spec, networkNamespace(spec))
		_, err := s.K8sClient.CoreV1().Namespaces().Create(ns)
		if apierrors.IsAlreadyExists(err) {
			s.Logger.Log("debug", "namespace "+ns.Name+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else if err != nil {
			return microerror.MaskAnyf(err, "creating namespace %s", ns.Name)
		}
	}

	// Create a dameonset running flanneld and creating network bridge.
	{
		daemonSet := newDaemonSet(spec)
		_, err := s.K8sClient.ExtensionsV1beta1().DaemonSets(networkNamespace(spec)).Create(daemonSet)
		if apierrors.IsAlreadyExists(err) {
			s.Logger.Log("debug", "daemonSet "+daemonSet.Name+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else if err != nil {
			return microerror.MaskAnyf(err, "creating daemonSet %s", daemonSet.Name)
		}
	}

	s.Logger.Log("info", "started flanneld", "event", "add", "cluster", spec.Cluster.ID)
	return nil
}

// deleteFunc waits for the delting cluster's namespace to be fully deleted and
// then cleans up flannel bridges.
func (s *Service) deleteFunc(obj interface{}) {
	defer func() {
		if r := recover(); r != nil {
			s.Logger.Log("error", "recovered in deleteFunc", "panic", r, "event", "delete")
		}
	}()
	err := s.deleteFuncError(obj)
	if err != nil {
		s.Logger.Log("error", fmt.Sprintf("%#v", err), "event", "delete")
	}
}

func (s *Service) deleteFuncError(obj interface{}) error {
	var spec flanneltpr.Spec
	{
		o, ok := obj.(*flanneltpr.CustomObject)
		if !ok {
			return microerror.MaskAnyf(wrongTypeError, "expected '%T', got '%T'", &flanneltpr.CustomObject{}, obj)
		}
		spec = o.Spec
	}

	waitForNamespaceDeleted := func(name string) error {
		// op does not mask errors, they are used only to be logged in notify.
		op := func() error {
			_, err := s.K8sClient.CoreV1().Namespaces().Get(name, apismetav1.GetOptions{})
			if err != nil && apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			return errors.New("still exists")
		}

		notify := func(reason error, interval time.Duration) {
			s.Logger.Log("debug", "waiting for the namespace "+name+" to be removed, reason: "+reason.Error(), "cluster", spec.Cluster.ID)
		}

		err := backoff.RetryNotify(op, backoff.NewExponentialBackOff(), notify)
		if err != nil {
			return microerror.MaskAnyf(err, "failed waiting for the namespace %s to be deleted", name)
		}

		return nil
	}

	// Wait for the cluster's namespace to be deleted.
	{
		s.Logger.Log("debug", "waiting for the cluster namespace to be deleted", "cluster", spec.Cluster.ID)

		err := waitForNamespaceDeleted(spec.Cluster.Namespace)
		if err != nil {
			return microerror.MaskAny(err)
		}
	}

	// Delete flannel network namespace.
	{
		s.Logger.Log("debug", "deleting flannel network namespace", "cluster", spec.Cluster.ID)

		ns := networkNamespace(spec)

		err := s.K8sClient.CoreV1().Namespaces().Delete(ns, &apismetav1.DeleteOptions{})
		if err != nil {
			return microerror.MaskAnyf(err, "deleting namespace %s failed", ns)
		}

		err = waitForNamespaceDeleted(ns)
		if err != nil {
			return microerror.MaskAny(err)
		}
	}

	// Create namespace for the cleanup job.
	{
		ns := newNamespace(spec, destroyerNamespace(spec))
		_, err := s.K8sClient.CoreV1().Namespaces().Create(ns)
		if err != nil {
			return microerror.MaskAnyf(err, "creating namespace %s", ns.Name)
		}
	}

	// Schedule flannel resources cleanup on every node using anti affinity
	// with hostname topology.
	podAffinity := newPodAffinity(spec)

	var replicas int32
	{
		// All nodes are listed assuming that master nodes run kubelets.
		nodes, err := s.K8sClient.CoreV1().Nodes().List(apismetav1.ListOptions{})
		if err != nil {
			return microerror.MaskAnyf(err, "requesting cluster node list")
		}

		// Run only on scheduleable nodes.
		for _, n := range nodes.Items {
			if !n.Spec.Unschedulable {
				replicas++
			}
		}
	}

	// Create a bridge cleanup job.
	var jobName string
	{
		s.Logger.Log("debug", "creating network bridge cleanup job", "cluster", spec.Cluster.ID)

		job := newJob(spec, replicas)
		job.Spec.Template.Spec.Affinity = podAffinity

		_, err := s.K8sClient.BatchV1().Jobs(destroyerNamespace(spec)).Create(job)
		if err != nil {
			return microerror.MaskAnyf(err, "creating job %s", jobName)
		}
		s.Logger.Log("debug", fmt.Sprintf("network bridge cleanup scheduled on %d nodes", replicas), "cluster", spec.Cluster.ID)

		jobName = job.Name
	}

	// Wait for the cleanup job to complete.
	{
		s.Logger.Log("debug", "waiting for network bridge cleanup job to complete", "cluster", spec.Cluster.ID)

		// op does not mask errors, they are used only to be logged in notify.
		op := func() error {
			job, err := s.K8sClient.BatchV1().Jobs(destroyerNamespace(spec)).Get(jobName, apismetav1.GetOptions{})
			if err != nil {
				return microerror.MaskAnyf(err, "requesting get job %s", jobName)
			}
			if job.Status.Succeeded != replicas {
				return fmt.Errorf("progress %d/%d", job.Status.Succeeded, replicas)
			}
			s.Logger.Log("debug", fmt.Sprintf("network bridge cleanup finished on %d nodes", job.Status.Succeeded), "cluster", spec.Cluster.ID)
			return nil
		}

		notify := func(reason error, interval time.Duration) {
			s.Logger.Log("debug", "waiting for network bridge cleanup to complete, reason: "+reason.Error(), "cluster", spec.Cluster.ID)
		}

		err := backoff.RetryNotify(op, backoff.NewExponentialBackOff(), notify)
		if err != nil {
			return microerror.MaskAnyf(err, "waiting for pods to finish network bridge cleanup")
		}
	}

	// Cleanup etcd.
	{
		s.Logger.Log("debug", "removing flannel etcd config", "cluster", spec.Cluster.ID)

		path := etcdPath(spec)

		err := s.store.Delete(context.TODO(), path)
		if storage.IsNotFound(err) {
			s.Logger.Log("debug", fmt.Sprintf("etcd path '%s' not found", path), "cluster", spec.Cluster.ID)
		} else if err != nil {
			return microerror.MaskAnyf(err, "deleting etcd path %s", path)
		}
	}

	// The operator's resources cleanup.
	{
		s.Logger.Log("debug", "removing cleanup resources", "cluster", spec.Cluster.ID)

		ns := destroyerNamespace(spec)
		err := s.K8sClient.CoreV1().Namespaces().Delete(ns, &apismetav1.DeleteOptions{})
		if err != nil {
			return microerror.MaskAnyf(err, "deleting namespace %s", ns)
		}
	}

	s.Logger.Log("info", "finished flannel cleanup for cluster", "cluster", spec.Cluster.ID)
	return nil
}
