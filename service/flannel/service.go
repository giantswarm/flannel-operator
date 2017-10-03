package flannel

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/coreos/etcd/clientv3"
	"github.com/giantswarm/etcdstorage"
	"github.com/giantswarm/microerror"
	microtls "github.com/giantswarm/microkit/tls"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/microstorage"
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
	BackOff   backoff.BackOff
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
		BackOff:   nil,
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
	if config.BackOff == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.BackOff client must not be empty")
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "kubernetes client must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "logger must not be empty")
	}

	// Settings.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "viper must not be empty")
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
			return nil, microerror.Mask(err)
		}
	}

	var tlsConfig *tls.Config
	{
		rootCAs := []string{}
		{
			v := config.Viper.GetString(config.Flag.Service.Etcd.TLS.CAFile)
			if v != "" {
				rootCAs = []string{v}
			}
		}
		certFiles := microtls.CertFiles{
			RootCAs: rootCAs,
			Cert:    config.Viper.GetString(config.Flag.Service.Etcd.TLS.CrtFile),
			Key:     config.Viper.GetString(config.Flag.Service.Etcd.TLS.KeyFile),
		}

		tlsConfig, err = microtls.LoadTLSConfig(certFiles)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var etcdClient *clientv3.Client
	{
		etcdConfig := clientv3.Config{
			Endpoints: []string{
				config.Viper.GetString(config.Flag.Service.Etcd.Endpoint),
			},
			TLS: tlsConfig,
		}
		etcdClient, err = clientv3.New(etcdConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var store microstorage.Storage
	{
		config := etcdstorage.DefaultConfig()

		config.EtcdClient = etcdClient

		store, err = etcdstorage.New(config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	newService := &Service{
		Config: config,

		bootOnce:    sync.Once{},
		etcdCAFile:  config.Viper.GetString(config.Flag.Service.Etcd.TLS.CAFile),
		etcdCrtFile: config.Viper.GetString(config.Flag.Service.Etcd.TLS.CrtFile),
		etcdKeyFile: config.Viper.GetString(config.Flag.Service.Etcd.TLS.KeyFile),
		store:       store,
		tpr:         newTPR,
	}

	return newService, nil
}

// Service implements the Flannel service.
type Service struct {
	Config

	bootOnce    sync.Once
	etcdCAFile  string
	etcdCrtFile string
	etcdKeyFile string
	store       microstorage.Storage
	tpr         *tpr.TPR
}

// Boot starts the service and implements the watch for the flannel TPR.
func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		o := func() error {
			err := s.bootWithError()
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}

		n := func(err error, d time.Duration) {
			s.Logger.Log("warning", fmt.Sprintf("retrying operator boot due to error: %#v", microerror.Mask(err)))
		}

		err := backoff.RetryNotify(o, s.BackOff, n)
		if err != nil {
			s.Logger.Log("error", fmt.Sprintf("stop operator boot retries due to too many errors: %#v", microerror.Mask(err)))
			os.Exit(1)
		}
	})
}

func (s *Service) bootWithError() error {
	err := s.tpr.CreateAndWait()
	if tpr.IsAlreadyExists(err) {
		s.Logger.Log("debug", "third party resource already exists")
	} else if err != nil {
		return microerror.Mask(err)
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

	return nil
}

// addFunc creates flannel etcd configuration, schedules flanneld container in
// daemonset and creates network bridge configuration for the cluster.
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
			return microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &flanneltpr.CustomObject{}, obj)
		}
		spec = o.Spec
	}

	// Create flannel etcd config.
	{
		k := etcdKey(spec)

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
			Network:   spec.Flannel.Spec.Network,
			SubnetLen: spec.Flannel.Spec.SubnetLen,
			Backend: flannelBackend{
				Type: "vxlan",
				VNI:  spec.Flannel.Spec.VNI,
			},
		}

		bytes, err := json.Marshal(config)
		if err != nil {
			return microerror.Maskf(err, "marshaling %#v", config)
		}

		exists, err := s.store.Exists(context.TODO(), k)
		if err != nil {
			return microerror.Maskf(err, "checking %s etcd key existence", k.Key())
		}
		if exists {
			s.Logger.Log("debug", "etcd key "+k.Key()+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else {
			kv := etcdKeyValue(spec, string(bytes))
			err := s.store.Put(context.TODO(), kv)
			if err != nil {
				return microerror.Maskf(err, "createing %s etcd key", k.Key())
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
			return microerror.Maskf(err, "creating namespace %s", ns.Name)
		}
	}

	// Create a dameonset running flanneld and creating network bridge.
	{
		daemonSet := newDaemonSet(spec, s.etcdCAFile, s.etcdCrtFile, s.etcdKeyFile)
		_, err := s.K8sClient.ExtensionsV1beta1().DaemonSets(networkNamespace(spec)).Create(daemonSet)
		if apierrors.IsAlreadyExists(err) {
			s.Logger.Log("debug", "daemonSet "+daemonSet.Name+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else if err != nil {
			return microerror.Maskf(err, "creating daemonSet %s", daemonSet.Name)
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
			return microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &flanneltpr.CustomObject{}, obj)
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
			return microerror.Maskf(err, "failed waiting for the namespace %s to be deleted", name)
		}

		return nil
	}

	// Wait for the cluster's namespace to be deleted.
	{
		s.Logger.Log("debug", "waiting for the cluster namespace to be deleted", "cluster", spec.Cluster.ID)

		err := waitForNamespaceDeleted(spec.Cluster.Namespace)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Delete flannel network namespace.
	{
		s.Logger.Log("debug", "deleting flannel network namespace", "cluster", spec.Cluster.ID)

		ns := networkNamespace(spec)

		err := s.K8sClient.CoreV1().Namespaces().Delete(ns, &apismetav1.DeleteOptions{})
		if err != nil {
			return microerror.Maskf(err, "deleting namespace %s failed", ns)
		}

		err = waitForNamespaceDeleted(ns)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create namespace for the cleanup job.
	{
		ns := newNamespace(spec, destroyerNamespace(spec))
		_, err := s.K8sClient.CoreV1().Namespaces().Create(ns)
		if err != nil {
			return microerror.Maskf(err, "creating namespace %s", ns.Name)
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
			return microerror.Maskf(err, "requesting cluster node list")
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
			return microerror.Maskf(err, "creating job %s", jobName)
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
				return microerror.Maskf(err, "requesting get job %s", jobName)
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
			return microerror.Maskf(err, "waiting for pods to finish network bridge cleanup")
		}
	}

	// Cleanup etcd.
	{
		s.Logger.Log("debug", "removing flannel etcd config", "cluster", spec.Cluster.ID)

		k := etcdKey(spec)

		err := s.store.Delete(context.TODO(), k)
		if microstorage.IsNotFound(err) {
			s.Logger.Log("debug", fmt.Sprintf("etcd key '%s' not found", k.Key()), "cluster", spec.Cluster.ID)
		} else if err != nil {
			return microerror.Maskf(err, "deleting etcd key %s", k.Key())
		}
	}

	// The operator's resources cleanup.
	{
		s.Logger.Log("debug", "removing cleanup resources", "cluster", spec.Cluster.ID)

		ns := destroyerNamespace(spec)
		err := s.K8sClient.CoreV1().Namespaces().Delete(ns, &apismetav1.DeleteOptions{})
		if err != nil {
			return microerror.Maskf(err, "deleting namespace %s", ns)
		}
	}

	s.Logger.Log("info", "finished flannel cleanup for cluster", "cluster", spec.Cluster.ID)
	return nil
}
