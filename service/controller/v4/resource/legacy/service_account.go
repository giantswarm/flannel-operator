package legacy

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	api "k8s.io/api/core/v1"
	apismeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
)

func newServiceAccount(customObject v1alpha1.FlannelConfig, name string) *api.ServiceAccount {
	serviceAccount := &api.ServiceAccount{
		TypeMeta: apismeta.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: apismeta.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app":         networkApp,
				"cluster-id":  key.ClusterID(customObject),
				"customer-id": key.ClusterCustomer(customObject),
			},
		},
	}

	return serviceAccount
}
