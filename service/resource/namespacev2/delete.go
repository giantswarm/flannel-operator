package namespacev2

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/framework"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/flannel-operator/service/keyv2"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	customObject, err := keyv2.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	namespaceToDelete, err := toNamespace(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if namespaceToDelete != nil {
		r.logger.Log("cluster", keyv2.ClusterID(customObject), "debug", "deleting the namespace in the Kubernetes API")

		err = r.k8sClient.CoreV1().Namespaces().Delete(namespaceToDelete.Name, &apismetav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Log("cluster", keyv2.ClusterID(customObject), "debug", "deleted the namespace in the Kubernetes API")
	} else {
		r.logger.Log("cluster", keyv2.ClusterID(customObject), "debug", "the namespace does not need to be deleted from the Kubernetes API")
	}

	return nil
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	delete, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := framework.NewPatch()
	patch.SetDeleteChange(delete)

	return patch, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	customObject, err := keyv2.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	currentNamespace, err := toNamespace(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredNamespace, err := toNamespace(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Log("cluster", keyv2.ClusterID(customObject), "debug", "finding out if the namespace has to be deleted")

	var namespaceToDelete *apiv1.Namespace
	if currentNamespace != nil {
		namespaceToDelete = desiredNamespace
	}

	r.logger.Log("cluster", keyv2.ClusterID(customObject), "debug", "found out if the namespace has to be deleted")

	return namespaceToDelete, nil
}
