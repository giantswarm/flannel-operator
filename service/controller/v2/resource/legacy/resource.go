package legacy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/flannel-operator/service/controller/v2/key"
)

const (
	// Name is the identifier of the resource.
	Name = "legacyv2"
)

// Config represents the configuration used to create a new config map resource.
type Config struct {
	BackOff   backoff.Interface
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
	backOff   backoff.Interface
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
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Log("cluster", key.ClusterID(customObject), "debug", "looking for the daemon set in the Kubernetes API")

	var currentDaemonSet *v1beta1.DaemonSet
	{
		manifest, err := r.k8sClient.Extensions().DaemonSets(networkNamespace(customObject.Spec)).Get(networkApp, apismetav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.Log("cluster", key.ClusterID(customObject), "debug", "did not find the daemon set in the Kubernetes API")
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			r.logger.Log("cluster", key.ClusterID(customObject), "debug", "found the daemon set in the Kubernetes API")
			currentDaemonSet = manifest
			r.updateVersionBundleVersionGauge(customObject, versionBundleVersionGauge, currentDaemonSet)
		}
	}

	return currentDaemonSet, nil
}

func (r *Resource) updateVersionBundleVersionGauge(customObject v1alpha1.FlannelConfig, gauge *prometheus.GaugeVec, daemonSet *v1beta1.DaemonSet) {
	version, ok := daemonSet.Annotations[VersionBundleVersionAnnotation]
	if !ok {
		r.logger.Log("cluster", key.ClusterID(customObject), "warning", fmt.Sprintf("cannot update current version bundle version metric: annotation '%s' must not be empty", VersionBundleVersionAnnotation))
		return
	}

	split := strings.Split(version, ".")
	if len(split) != 3 {
		r.logger.Log("cluster", key.ClusterID(customObject), "warning", fmt.Sprintf("cannot update current version bundle version metric: invalid version format, expected '<major>.<minor>.<patch>', got '%s'", version))
		return
	}

	major := split[0]
	minor := split[1]
	patch := split[2]

	gauge.WithLabelValues(major, minor, patch).Set(1)
}

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	return nil, nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Create a service account for the daemonset
	{
		serviceAccount := newServiceAccount(customObject, serviceAccountName(customObject.Spec))
		_, err := r.k8sClient.CoreV1().ServiceAccounts(networkNamespace(customObject.Spec)).Create(serviceAccount)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "serviceAccount "+serviceAccount.Name+" already exists", "event", "add", "cluster", customObject.Spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Maskf(err, "creating serviceAccount %s", serviceAccount.Name)
		}
	}

	// Bind the service account with the cluster role of flannel operator
	{
		clusterRoleBinding := newClusterRoleBinding(customObject)
		_, err := r.k8sClient.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "clusterRoleBinding "+clusterRoleBinding.Name+" already exists", "event", "add", "cluster", customObject.Spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Maskf(err, "creating clusterRoleBinding %s", clusterRoleBinding.Name)
		}
	}

	// Bind the service account with the cluster role of flannel operator pod security policy
	{
		clusterRoleBindingPodSecurityPolicy := newClusterRoleBindingPodSecurityPolicy(customObject)
		_, err := r.k8sClient.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBindingPodSecurityPolicy)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "clusterRoleBindingPodSecurityPolicy "+clusterRoleBindingPodSecurityPolicy.Name+" already exists", "event", "add", "cluster", customObject.Spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Maskf(err, "creating clusterRoleBindingPodSecurityPolicy %s", clusterRoleBindingPodSecurityPolicy.Name)
		}
	}

	// Create a dameonset running flanneld and creating network bridge.
	{
		daemonSet := newDaemonSet(customObject, r.etcdCAFile, r.etcdCrtFile, r.etcdKeyFile)
		_, err := r.k8sClient.ExtensionsV1beta1().DaemonSets(networkNamespace(customObject.Spec)).Create(daemonSet)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "daemonSet "+daemonSet.Name+" already exists", "event", "add", "cluster", customObject.Spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Maskf(err, "creating daemonSet %s", daemonSet.Name)
		}
	}

	r.logger.Log("info", "started flanneld", "event", "add", "cluster", customObject.Spec.Cluster.ID)

	return nil, nil
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	delete, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := controller.NewPatch()
	patch.SetDeleteChange(delete)

	return patch, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	spec := customObject.Spec

	// Delete the service account for the daemonset
	{
		serviceAccountName := serviceAccountName(customObject.Spec)
		err := r.k8sClient.CoreV1().ServiceAccounts(networkNamespace(customObject.Spec)).Delete(serviceAccountName, &apismetav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Maskf(err, "deleting service account %s", serviceAccountName)
		}
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

		err := backoff.RetryNotify(op, backoff.NewExponential(15*time.Minute, 5*time.Minute), notify)
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
		if apierrors.IsAlreadyExists(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Bind the service account for the clean up with the cluster role of flannel operator
	{
		clusterRoleBinding := newClusterRoleBindingForDeletion(customObject)
		_, err := r.k8sClient.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "clusterRoleBinding "+clusterRoleBinding.Name+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Maskf(err, "creating clusterRoleBinding %s", clusterRoleBinding.Name)
		}
	}

	// Bind the service account for the clean up with the cluster role of flannel operator psp
	{
		clusterRoleBinding := newClusterRoleBindingPodSecurityPolicyForDeletion(customObject)
		_, err := r.k8sClient.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "clusterRoleBinding "+clusterRoleBinding.Name+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Maskf(err, "creating clusterRoleBinding %s", clusterRoleBinding.Name)
		}
	}

	// Create a service account for the cleanup job.
	{
		serviceAccount := newServiceAccount(customObject, key.ClusterID(customObject))
		_, err := r.k8sClient.CoreV1().ServiceAccounts(destroyerNamespace(spec)).Create(serviceAccount)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "serviceAccount "+serviceAccount.Name+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Maskf(err, "creating serviceAccount %s", serviceAccount.Name)
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
		if apierrors.IsAlreadyExists(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
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

		err := backoff.RetryNotify(op, backoff.NewExponential(15*time.Minute, 5*time.Minute), notify)
		if err != nil {
			return nil, microerror.Maskf(err, "waiting for pods to finish network bridge cleanup")
		}
	}

	// The operator's resources cleanup.
	{
		r.logger.Log("debug", "removing cleanup resources", "cluster", spec.Cluster.ID)

		ns := destroyerNamespace(spec)
		err := r.k8sClient.CoreV1().Namespaces().Delete(ns, &apismetav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Remove cluster role bindings.
	{
		r.logger.Log("debug", "removing cluster role bindings", "cluster", spec.Cluster.ID)

		clusterRoleBindingForDeletionName := clusterRoleBindingForDeletion(spec)
		err = r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingForDeletionName, &apismetav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		clusterRoleBindingName := clusterRoleBinding(spec)
		err := r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingName, &apismetav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		clusterRoleBindingForPodSecurityPolicyName := clusterRoleBindingForPodSecurityPolicy(spec)
		err = r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingForPodSecurityPolicyName, &apismetav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		clusterRoleBindingForPodSecurityPolicyForDeletionName := clusterRoleBindingForPodSecurityPolicyForDeletion(spec)
		err = r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingForPodSecurityPolicyForDeletionName, &apismetav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r.logger.Log("info", "finished flannel cleanup for cluster", "cluster", spec.Cluster.ID)

	return nil, nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	create, err := r.newCreateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := controller.NewPatch()
	patch.SetCreateChange(create)
	patch.SetUpdateChange(update)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	return nil, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	return nil
}

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	return nil
}

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	return nil
}
