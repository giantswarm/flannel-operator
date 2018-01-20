package namespacev2

import (
	"context"

	"github.com/giantswarm/microerror"
	apiv1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/flannel-operator/service/keyv2"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := keyv2.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Log("cluster", keyv2.ClusterID(customObject), "debug", "computing the desired namespace")

	namespace := &apiv1.Namespace{
		TypeMeta: apismetav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: apismetav1.ObjectMeta{
			Name: keyv2.NetworkNamespace(customObject),
			Labels: map[string]string{
				"cluster-id":  keyv2.ClusterID(customObject),
				"customer-id": keyv2.ClusterCustomer(customObject),
			},
		},
	}

	r.logger.Log("cluster", keyv2.ClusterID(customObject), "debug", "computed the desired namespace")

	return namespace, nil
}
