package legacy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/finalizerskeptcontext"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	"github.com/giantswarm/operatorkit/resource/crud"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
)

const (
	// Name is the identifier of the resource.
	Name = "legacyv3"
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
	return nil, nil
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
		_, err := r.k8sClient.CoreV1().ServiceAccounts(key.NetworkNamespace(customObject)).Create(serviceAccount)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "serviceAccount "+serviceAccount.Name+" already exists", "event", "add", "cluster", customObject.Spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Bind the service account with the cluster role of flannel operator
	{
		clusterRoleBinding := newClusterRoleBinding(customObject)
		_, err := r.k8sClient.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "clusterRoleBinding "+clusterRoleBinding.Name+" already exists", "event", "add", "cluster", customObject.Spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Bind the service account with the cluster role of flannel operator pod security policy
	{
		clusterRoleBindingPodSecurityPolicy := newClusterRoleBindingPodSecurityPolicy(customObject)
		_, err := r.k8sClient.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBindingPodSecurityPolicy)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "clusterRoleBindingPodSecurityPolicy "+clusterRoleBindingPodSecurityPolicy.Name+" already exists", "event", "add", "cluster", customObject.Spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r.logger.Log("info", "started flanneld", "event", "add", "cluster", customObject.Spec.Cluster.ID)

	return nil, nil
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	delete, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
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
	list, err := r.k8sClient.CoreV1().Pods(n).List(metav1.ListOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(list.Items) != 0 {
		r.logger.LogCtx(ctx, "level", "debug", "message", "cannot finish deletion of network due to existing pods")

		finalizerskeptcontext.SetKept(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "keeping finalizers")

		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

		return nil, nil
	}

	// We delete extensions/v1beta1 daemon sets we find. They were once managed
	// with the legacy resource implementation. The new approach is apps/v1 daemon
	// sets managed by the flanneld resource implementation. When there is no
	// daemon set to delete here, the other resource implementation will take
	// over.
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting the legacy daemon set in the Kubernetes API")

		b := metav1.DeletePropagationBackground
		o := &metav1.DeleteOptions{
			PropagationPolicy: &b,
		}

		err := r.k8sClient.ExtensionsV1beta1().DaemonSets(key.NetworkNamespace(customObject)).Delete(key.NetworkID, o)
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "deleted the legacy daemon set in the Kubernetes API")
	}

	// Delete the service account for the daemonset
	{
		serviceAccountName := serviceAccountName(customObject.Spec)
		err := r.k8sClient.CoreV1().ServiceAccounts(key.NetworkNamespace(customObject)).Delete(serviceAccountName, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	waitForNamespaceDeleted := func(name string) error {
		// op does not mask errors, they are used only to be logged in notify.
		op := func() error {
			_, err := r.k8sClient.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
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

		err := backoff.RetryNotify(op, backoff.NewExponential(2*time.Minute, 5*time.Second), notify)
		if err != nil {
			return microerror.Mask(err)
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

		err := waitForNamespaceDeleted(key.NetworkNamespace(customObject))
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Create namespace for the cleanup job.
	{
		ns := newNamespace(customObject, destroyerNamespace(spec))
		_, err := r.k8sClient.CoreV1().Namespaces().Create(ns)
		if apierrors.IsAlreadyExists(err) {
			namespace, err := r.k8sClient.CoreV1().Namespaces().Get(ns.GetName(), metav1.GetOptions{})
			if err != nil {
				return nil, microerror.Mask(err)
			}

			if namespace != nil && namespace.Status.Phase == "Terminating" {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("destroyer namespace is in phase %#q", namespace.Status.Phase))

				finalizerskeptcontext.SetKept(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "keeping finalizers")

				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

				return nil, nil
			}
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
			return nil, microerror.Mask(err)
		}
	}

	// Bind the service account for the clean up with the cluster role of flannel operator psp
	{
		clusterRoleBinding := newClusterRoleBindingPodSecurityPolicyForDeletion(customObject)
		_, err := r.k8sClient.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "clusterRoleBinding "+clusterRoleBinding.Name+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Create a service account for the cleanup job.
	{
		serviceAccount := newServiceAccount(customObject, key.ClusterID(customObject))
		_, err := r.k8sClient.CoreV1().ServiceAccounts(destroyerNamespace(spec)).Create(serviceAccount)
		if apierrors.IsAlreadyExists(err) {
			r.logger.Log("debug", "serviceAccount "+serviceAccount.Name+" already exists", "event", "add", "cluster", spec.Cluster.ID)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Schedule flannel resources cleanup on every node using anti affinity
	// with hostname topology.
	podAffinity := newPodAffinity(spec)

	var replicas int32
	{
		// All nodes are listed assuming that master nodes run kubelets.
		nodes, err := r.k8sClient.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return nil, microerror.Mask(err)
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

		job := newJob(customObject, replicas)
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
			job, err := r.k8sClient.BatchV1().Jobs(destroyerNamespace(spec)).Get(jobName, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
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

		err := backoff.RetryNotify(op, backoff.NewExponential(2*time.Minute, 5*time.Second), notify)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// The operator's resources cleanup.
	{
		r.logger.Log("debug", "removing cleanup resources", "cluster", spec.Cluster.ID)

		ns := destroyerNamespace(spec)
		err := r.k8sClient.CoreV1().Namespaces().Delete(ns, &metav1.DeleteOptions{})
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
		err = r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingForDeletionName, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		clusterRoleBindingName := clusterRoleBinding(spec)
		err := r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingName, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		clusterRoleBindingForPodSecurityPolicyName := clusterRoleBindingForPodSecurityPolicy(spec)
		err = r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingForPodSecurityPolicyName, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		clusterRoleBindingForPodSecurityPolicyForDeletionName := clusterRoleBindingForPodSecurityPolicyForDeletion(spec)
		err = r.k8sClient.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingForPodSecurityPolicyForDeletionName, &metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r.logger.Log("info", "finished flannel cleanup for cluster", "cluster", spec.Cluster.ID)

	return nil, nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	create, err := r.newCreateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
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
