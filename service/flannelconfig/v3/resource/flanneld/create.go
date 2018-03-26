package flanneld

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/flannel-operator/service/flannelconfig/v3/key"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	daemonSetToCreate, err := toDaemonSet(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if daemonSetToCreate != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", "creating the daemon set in the Kubernetes API")

		namespace := key.NetworkNamespace(customObject)
		_, err = r.k8sClient.Extensions().DaemonSets(namespace).Create(daemonSetToCreate)
		if apierrors.IsAlreadyExists(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "created the daemon set in the Kubernetes API")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the daemon set does not need to be created in the Kubernetes API")
	}

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentDaemonSet, err := toDaemonSet(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredDaemonSet, err := toDaemonSet(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if the daemon set has to be created")

	var daemonSetToCreate *v1beta1.DaemonSet
	if currentDaemonSet == nil {
		daemonSetToCreate = desiredDaemonSet
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found out if the daemon set has to be created")

	return daemonSetToCreate, nil
}
