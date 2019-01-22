package legacy

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
)

// newNamespace creates a namespace with a given name. The created namespace
// has a commont set of labels for this operator.
func newNamespace(customObject v1alpha1.FlannelConfig, name string) *apiv1.Namespace {
	return &apiv1.Namespace{
		TypeMeta: apismetav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: apismetav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"cluster":  key.ClusterID(customObject),
				"customer": key.ClusterCustomer(customObject),
			},
		},
	}
}
