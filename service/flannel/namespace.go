package flannel

import (
	"github.com/giantswarm/flanneltpr"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func newNamespace(spec flanneltpr.Spec) *apiv1.Namespace {
	return &apiv1.Namespace{
		TypeMeta: apismetav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: apismetav1.ObjectMeta{
			Name: destroyerNamespace(spec),
			Labels: map[string]string{
				"cluster":  clusterName(spec),
				"customer": clusterCustomer(spec),
			},
		},
	}
}
