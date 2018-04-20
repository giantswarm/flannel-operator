package legacy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/controller/context/finalizerskeptcontext"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
)

const (
	// Name is the identifier of the resource.
	Name = "legacyv3"
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
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the daemon set in the Kubernetes API")

	var currentDaemonSet *v1beta1.DaemonSet
	{
		manifest, err := r.k8sClient.Extensions().DaemonSets(networkNamespace(customObject.Spec)).Get(networkApp, apismetav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the daemon set in the Kubernetes API")
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "found the daemon set in the Kubernetes API")
			currentDaemonSet = manifest
			r.updateVersionBundleVersionGauge(ctx, customObject, versionBundleVersionGauge, currentDaemonSet)
		}
	}

	return currentDaemonSet, nil
}

func (r *Resource) updateVersionBundleVersionGauge(ctx context.Context, customObject v1alpha1.FlannelConfig, gauge *prometheus.GaugeVec, daemonSet *v1beta1.DaemonSet) {
	version, ok := daemonSet.Annotations[VersionBundleVersionAnnotation]
	if !ok {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("cannot update current version bundle version metric: annotation '%s' must not be empty", VersionBundleVersionAnnotation))
		return
	}

	split := strings.Split(version, ".")
	if len(split) != 3 {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("cannot update current version bundle version metric: invalid version format, expected '<major>.<minor>.<patch>', got '%s'", version))
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

	// In case a cluster deletion happens, we want to delete the guest cluster
	// network. We still need to use the network for resource creation in
	// order to drain nodes on KVM though. So as long as pods are there we delay
	// the deletion of the network here in order to still be able to create
	// resources. As soon as the draining was done and the pods got removed
	// we get an empty list here after the delete event got replayed. Then we just
	// remove the namespace as usual.
	n := key.ClusterNamespace(customObject)
	list, err := r.k8sClient.CoreV1().Pods(n).List(apismetav1.ListOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(list.Items) != 0 {
		r.logger.LogCtx(ctx, "level", "debug", "message", "cannot finish deletion of network due to existing pods")
		finalizerskeptcontext.SetKept(ctx)
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

		return nil, nil
	}

	// Delete the service account for the daemonset
	{
		serviceAccountName := serviceAccountName(customObject.Spec)
		err := r.k8sClient.CoreV1().ServiceAccounts(networkNamespace(customObject.Spec)).Delete(serviceAccountName, &apismetav1.DeleteOptions{})
		if err != nil {
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

		err := backoff.RetryNotify(op, NewExponentialBackoff(ShortMaxWait, ShortMaxInterval), notify)
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

		err := backoff.RetryNotify(op, NewExponentialBackoff(ShortMaxWait, ShortMaxInterval), notify)
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

	// Remove cluster role bindings.
	{
		r.logger.Log("debug", "removing cluster role bindings", "cluster", spec.Cluster.ID)

		clusterRoleBindingForDeletionName := clusterRoleBindingForDeletion(spec)
		err = r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingForDeletionName, &apismetav1.DeleteOptions{})
		if err != nil {
			return nil, microerror.Maskf(err, "deleting cluster role binding %s", clusterRoleBindingForDeletionName)
		}

		clusterRoleBindingName := clusterRoleBinding(spec)
		err := r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingName, &apismetav1.DeleteOptions{})
		if err != nil {
			return nil, microerror.Maskf(err, "deleting cluster role binding %s", clusterRoleBindingName)
		}

		clusterRoleBindingForPodSecurityPolicyName := clusterRoleBindingForPodSecurityPolicy(spec)
		err = r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingForPodSecurityPolicyName, &apismetav1.DeleteOptions{})
		if err != nil {
			return nil, microerror.Maskf(err, "deleting cluster role binding %s", clusterRoleBindingForPodSecurityPolicyName)
		}

		clusterRoleBindingForPodSecurityPolicyForDeletionName := clusterRoleBindingForPodSecurityPolicyForDeletion(spec)
		err = r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingForPodSecurityPolicyForDeletionName, &apismetav1.DeleteOptions{})
		if err != nil {
			return nil, microerror.Maskf(err, "deleting cluster role binding %s", clusterRoleBindingForPodSecurityPolicyForDeletionName)
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
