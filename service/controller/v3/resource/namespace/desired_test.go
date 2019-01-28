package namespace

import (
	"context"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_Resource_Namespace_GetDesiredState(t *testing.T) {
	testCases := []struct {
		Obj          interface{}
		ExpectedName string
	}{
		{
			Obj: &v1alpha1.FlannelConfig{
				Spec: v1alpha1.FlannelConfigSpec{
					Cluster: v1alpha1.FlannelConfigSpecCluster{
						ID: "al9qy",
					},
				},
			},
			ExpectedName: "flannel-network-al9qy",
		},
		{
			Obj: &v1alpha1.FlannelConfig{
				Spec: v1alpha1.FlannelConfigSpec{
					Cluster: v1alpha1.FlannelConfigSpecCluster{
						ID: "foobar",
					},
				},
			},
			ExpectedName: "flannel-network-foobar",
		},
	}

	var err error
	var newResource *Resource
	{
		c := Config{
			K8sClient: fake.NewSimpleClientset(),
			Logger:    microloggertest.New(),
		}

		newResource, err = New(c)
		if err != nil {
			t.Fatal("expected", nil, "got", err)
		}
	}

	for i, tc := range testCases {
		result, err := newResource.GetDesiredState(context.TODO(), tc.Obj)
		if err != nil {
			t.Fatal("case", i+1, "expected", nil, "got", err)
		}
		name := result.(*apiv1.Namespace).Name
		if tc.ExpectedName != name {
			t.Fatalf("case %d expected %#v got %#v", i+1, tc.ExpectedName, name)
		}
	}
}
