package legacy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenk/backoff"
	"github.com/giantswarm/flanneltpr"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// Name is the identifier of the resource.
	Name = "legacy"
)

// Config represents the configuration used to create a new config map resource.
type Config struct {
	BackOff   backoff.BackOff
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	EtcdCAFile  string
	EtcdCrtFile string
	EtcdKeyFile string
}

// DefaultConfig provides a default configuration to create a new config map
// resource by best effort.
func DefaultConfig() Config {
	return Config{
		BackOff:   nil,
		K8sClient: nil,
		Logger:    nil,

		EtcdCAFile:  "",
		EtcdCrtFile: "",
		EtcdKeyFile: "",
	}
}

// Resource implements the config map resource.
type Resource struct {
	backOff   backoff.BackOff
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	etcdCAFile  string
	etcdCrtFile string
	etcdKeyFile string
}

// New creates a new configured config map resource.
func New(config Config) (*Resource, error) {
	if config.BackOff == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.BackOff must not be empty")
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sClient must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	newResource := &Resource{
		backOff:   config.BackOff,
		k8sClient: config.K8sClient,
		logger: config.Logger.With(
			"resource", Name,
		),

		etcdCAFile:  config.EtcdCAFile,
		etcdCrtFile: config.EtcdCrtFile,
		etcdKeyFile: config.EtcdKeyFile,
	}

	return newResource, nil
}

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	return nil, nil
}

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	return nil, nil
}

func (r *Resource) GetCreateState(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	var spec flanneltpr.Spec
	{
		o, ok := obj.(*flanneltpr.CustomObject)
		if !ok {
			return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &flanneltpr.CustomObject{}, obj)
		}
		spec = o.Spec
	}

	// Create a dameonset running flanneld and creating network bridge.
	{
		daemonSet := newDaemonSet(spec, r.etcdCAFile, r.etcdCrtFile, r.etcdKeyFile)
		_, err := r.k8sClient.ExtensionsV1beta1().DaemonSets(networkNamespace(spec)).Create(daemonSet)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "daemonSet "+daemonSet.Name+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Maskf(err, "creating daemonSet %s", daemonSet.Name)
		}
	}

	r.logger.Log("info", "started flanneld", "event", "add", "cluster", spec.Cluster.ID)

	return nil, nil
}

func (r *Resource) GetDeleteState(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	var spec flanneltpr.Spec
	{
		o, ok := obj.(*flanneltpr.CustomObject)
		if !ok {
			return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &flanneltpr.CustomObject{}, obj)
		}
		spec = o.Spec
	}

	waitForNamespaceDeleted := func(name string) error {
		// op does not mask errors, they are used only to be logged in notify.
		op := func() error {
			_, err := r.k8sClient.CoreV1().Namespaces().Get(name, apismetav1.GetOptions{})
			if err != nil && apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			return errors.New("still exists")
		}

		notify := func(reason error, interval time.Duration) {
			r.logger.Log("debug", "waiting for the namespace "+name+" to be removed, reason: "+reason.Error(), "cluster", spec.Cluster.ID)
		}

		err := backoff.RetryNotify(op, backoff.NewExponentialBackOff(), notify)
		if err != nil {
			return microerror.Maskf(err, "failed waiting for the namespace %s to be deleted", name)
		}

		return nil
	}

	{
		r.logger.Log("debug", "waiting for the cluster namespace to be deleted", "cluster", spec.Cluster.ID)

		err := waitForNamespaceDeleted(spec.Cluster.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		r.logger.Log("debug", "waiting for the flannel namespace to be deleted", "cluster", spec.Cluster.ID)

		err := waitForNamespaceDeleted(networkNamespace(spec))
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Create namespace for the cleanup job.
	{
		ns := newNamespace(spec, destroyerNamespace(spec))
		_, err := r.k8sClient.CoreV1().Namespaces().Create(ns)
		if err != nil {
			return nil, microerror.Maskf(err, "creating namespace %s", ns.Name)
		}
	}

	// Schedule flannel resources cleanup on every node using anti affinity
	// with hostname topology.
	podAffinity := newPodAffinity(spec)

	var replicas int32
	{
		// All nodes are listed assuming that master nodes run kubelets.
		nodes, err := r.k8sClient.CoreV1().Nodes().List(apismetav1.ListOptions{})
		if err != nil {
			return nil, microerror.Maskf(err, "requesting cluster node list")
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
		r.logger.Log("debug", "creating network bridge cleanup job", "cluster", spec.Cluster.ID)

		job := newJob(spec, replicas)
		job.Spec.Template.Spec.Affinity = podAffinity

		_, err := r.k8sClient.BatchV1().Jobs(destroyerNamespace(spec)).Create(job)
		if err != nil {
			return nil, microerror.Maskf(err, "creating job %s", jobName)
		}
		r.logger.Log("debug", fmt.Sprintf("network bridge cleanup scheduled on %d nodes", replicas), "cluster", spec.Cluster.ID)

		jobName = job.Name
	}

	// Wait for the cleanup job to complete.
	{
		r.logger.Log("debug", "waiting for network bridge cleanup job to complete", "cluster", spec.Cluster.ID)

		// op does not mask errors, they are used only to be logged in notify.
		op := func() error {
			job, err := r.k8sClient.BatchV1().Jobs(destroyerNamespace(spec)).Get(jobName, apismetav1.GetOptions{})
			if err != nil {
				return microerror.Maskf(err, "requesting get job %s", jobName)
			}
			if job.Status.Succeeded != replicas {
				return fmt.Errorf("progress %d/%d", job.Status.Succeeded, replicas)
			}
			r.logger.Log("debug", fmt.Sprintf("network bridge cleanup finished on %d nodes", job.Status.Succeeded), "cluster", spec.Cluster.ID)
			return nil
		}

		notify := func(reason error, interval time.Duration) {
			r.logger.Log("debug", "waiting for network bridge cleanup to complete, reason: "+reason.Error(), "cluster", spec.Cluster.ID)
		}

		err := backoff.RetryNotify(op, backoff.NewExponentialBackOff(), notify)
		if err != nil {
			return nil, microerror.Maskf(err, "waiting for pods to finish network bridge cleanup")
		}
	}

	// The operator's resources cleanup.
	{
		r.logger.Log("debug", "removing cleanup resources", "cluster", spec.Cluster.ID)

		ns := destroyerNamespace(spec)
		err := r.k8sClient.CoreV1().Namespaces().Delete(ns, &apismetav1.DeleteOptions{})
		if err != nil {
			return nil, microerror.Maskf(err, "deleting namespace %s", ns)
		}
	}

	r.logger.Log("info", "finished flannel cleanup for cluster", "cluster", spec.Cluster.ID)

	return nil, nil
}

func (r *Resource) GetUpdateState(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, interface{}, interface{}, error) {
	return nil, nil, nil, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) ProcessCreateState(ctx context.Context, obj, createState interface{}) error {
	return nil
}

func (r *Resource) ProcessDeleteState(ctx context.Context, obj, deleteState interface{}) error {
	return nil
}

func (r *Resource) ProcessUpdateState(ctx context.Context, obj, updateState interface{}) error {
	return nil
}

func (r *Resource) Underlying() framework.Resource {
	return r
}
