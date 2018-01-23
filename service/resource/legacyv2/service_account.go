package legacyv2

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	api "k8s.io/api/core/v1"
	apismeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newServiceAccount(customObject v1alpha1.FlannelConfig, name string) *api.ServiceAccount {
	app := networkApp

	serviceAccount := &api.ServiceAccount{
		TypeMeta: apismeta.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: apismeta.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app":         app,
				"cluster-id":  clusterName(customObject.Spec),
				"customer-id": clusterCustomer(customObject.Spec),
			},
		},
	}

	return serviceAccount
}
