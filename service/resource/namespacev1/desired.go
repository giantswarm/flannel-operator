package namespacev1

import (
	"context"

	"github.com/giantswarm/microerror"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"

	"github.com/giantswarm/flannel-operator/service/keyv1"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := keyv1.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Log("cluster", keyv1.ClusterID(customObject), "debug", "computing the desired namespace")

	namespace := &apiv1.Namespace{
		TypeMeta: apismetav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: apismetav1.ObjectMeta{
			Name: keyv1.NetworkNamespace(customObject),
			Labels: map[string]string{
				"cluster":  keyv1.ClusterID(customObject),
				"customer": keyv1.ClusterCustomer(customObject),
			},
		},
	}

	r.logger.Log("cluster", keyv1.ClusterID(customObject), "debug", "computed the desired namespace")

	return namespace, nil
}
