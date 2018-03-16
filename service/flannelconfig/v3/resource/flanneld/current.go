package flanneld

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/flannel-operator/service/flannelconfig/v3/key"
	"github.com/giantswarm/microerror"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "debug", "looking for the daemon set in the Kubernetes API")

	var currentDaemonSet *v1beta1.DaemonSet
	{
		namespace := key.NetworkNamespace(customObject)
		manifest, err := r.k8sClient.Extensions().DaemonSets(namespace).Get(key.NetworkID, apismetav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
			r.logger.LogCtx(ctx, "debug", "did not find the daemon set in the Kubernetes API")
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "debug", "found the daemon set in the Kubernetes API")

			currentDaemonSet = manifest

			r.updateVersionBundleVersionGauge(ctx, customObject, versionBundleVersionGauge, currentDaemonSet)
		}
	}

	return currentDaemonSet, nil
}

func (r *Resource) updateVersionBundleVersionGauge(ctx context.Context, customObject v1alpha1.FlannelConfig, gauge *prometheus.GaugeVec, daemonSet *v1beta1.DaemonSet) {
	version, ok := daemonSet.Annotations[VersionBundleVersionAnnotation]
	if !ok {
		r.logger.LogCtx(ctx, "warning", fmt.Sprintf("cannot update current version bundle version metric: annotation '%s' must not be empty", VersionBundleVersionAnnotation))
		return
	}

	split := strings.Split(version, ".")
	if len(split) != 3 {
		r.logger.LogCtx(ctx, "warning", fmt.Sprintf("cannot update current version bundle version metric: invalid version format, expected '<major>.<minor>.<patch>', got '%s'", version))
		return
	}

	major := split[0]
	minor := split[1]
	patch := split[2]

	gauge.WithLabelValues(major, minor, patch).Set(1)
}
