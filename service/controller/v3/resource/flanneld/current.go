package flanneld

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/finalizerskeptcontext"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	"github.com/prometheus/client_golang/prometheus"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// The flanneld resource implementation manages the apps/v1 daemon sets. The
	// legacy resource implementation manages extensions/v1beta1 daemon sets
	// still. In case we find the old daemon sets here, we cancel the resource and
	// let the other resource take over.
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the legacy daemon set in the Kubernetes API")

		namespace := key.NetworkNamespace(customObject)
		_, err = r.k8sClient.ExtensionsV1beta1().DaemonSets(namespace).Get(key.NetworkID, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the legacy daemon set in the Kubernetes API")
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "found the legacy daemon set in the Kubernetes API")

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			resourcecanceledcontext.SetCanceled(ctx)

			return nil, nil
		}
	}

	// In case a tenant cluster deletion happens, we want to delete the tenant
	// cluster network. We still need to use the network for resource creation in
	// order to drain nodes on KVM though. So as long as pods are there we delay
	// the deletion of the network here in order to still be able to create
	// resources. As soon as the draining was done and the pods got removed, we
	// get an empty list after the delete event got replayed. Then we delete the
	// daemon set as usual.
	if key.IsDeleted(customObject) {
		list, err := r.k8sClient.CoreV1().Pods(key.ClusterNamespace(customObject)).List(metav1.ListOptions{})
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(list.Items) != 0 {
			r.logger.LogCtx(ctx, "level", "debug", "message", "cannot finish deletion due to existing pods")

			finalizerskeptcontext.SetKept(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "keeping finalizers")

			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			return nil, nil
		}
	}

	var currentDaemonSet *appsv1.DaemonSet
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the daemon set in the Kubernetes API")

		namespace := key.NetworkNamespace(customObject)
		manifest, err := r.k8sClient.AppsV1().DaemonSets(namespace).Get(key.NetworkID, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the daemon set in the Kubernetes API")
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

func (r *Resource) updateVersionBundleVersionGauge(ctx context.Context, customObject v1alpha1.FlannelConfig, gauge *prometheus.GaugeVec, daemonSet *appsv1.DaemonSet) {
	version, ok := daemonSet.Annotations[VersionBundleVersionAnnotation]
	if !ok {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("cannot update current version bundle version metric: annotation %#q must not be empty", VersionBundleVersionAnnotation))
		return
	}

	split := strings.Split(version, ".")
	if len(split) != 3 {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("cannot update current version bundle version metric: invalid version format, expected '<major>.<minor>.<patch>', got %#q", version))
		return
	}

	major := split[0]
	minor := split[1]
	patch := split[2]

	gauge.WithLabelValues(major, minor, patch).Set(1)
}
