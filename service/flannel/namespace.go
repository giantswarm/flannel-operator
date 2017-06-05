package flannel

import (
	"github.com/giantswarm/flanneltpr"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
)

func newNamespace(spec flanneltpr.Spec) *v1.Namespace {
	return &v1.Namespace{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: destroyerNamespace(spec),
			Labels: map[string]string{
				"cluster":  clusterName(spec),
				"customer": clusterCustomer(spec),
			},
		},
	}
}
