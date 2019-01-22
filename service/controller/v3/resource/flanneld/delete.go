package flanneld

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	daemonSetToDelete, err := toDaemonSet(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if daemonSetToDelete != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting the daemon set in the Kubernetes API")

		name := daemonSetToDelete.GetName()
		namespace := daemonSetToDelete.GetNamespace()
		err := r.k8sClient.Extensions().DaemonSets(namespace).Delete(name, &metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "deleted the daemon set in the Kubernetes API")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "the daemon set does not need to be deleted in the Kubernetes API")
	}

	return nil
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
	currentDaemonSet, err := toDaemonSet(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return currentDaemonSet, nil
}
