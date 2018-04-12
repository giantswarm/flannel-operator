package networkconfigv2

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"

	etcdfake "github.com/giantswarm/flannel-operator/service/controller/v2/etcd/fake"
)

func Test_Resource_NetworkConfig_GetDesiredState(t *testing.T) {
	testCases := []struct {
		Obj                   interface{}
		ExpectedNetworkConfig NetworkConfig
	}{
		{
			Obj: &v1alpha1.FlannelConfig{},
			ExpectedNetworkConfig: NetworkConfig{
				Backend: Backend{
					Type: "vxlan",
				},
			},
		},

		{
			Obj: &v1alpha1.FlannelConfig{
				Spec: v1alpha1.FlannelConfigSpec{
					Flannel: v1alpha1.FlannelConfigSpecFlannel{
						Spec: v1alpha1.FlannelConfigSpecFlannelSpec{
							Network:   "172.26.0.0/16",
							SubnetLen: 30,
							VNI:       26,
						},
					},
				},
			},
			ExpectedNetworkConfig: NetworkConfig{
				Network:   "172.26.0.0/16",
				SubnetLen: 30,
				Backend: Backend{
					Type: "vxlan",
					VNI:  26,
				},
			},
		},
	}

	var err error
	var newResource *Resource
	{
		resourceConfig := DefaultConfig()

		resourceConfig.Logger = microloggertest.New()
		resourceConfig.Store = etcdfake.New()

		newResource, err = New(resourceConfig)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}
	}

	for _, tc := range testCases {
		desiredState, err := newResource.GetDesiredState(context.TODO(), tc.Obj)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		desiredNetworkConfig, ok := desiredState.(NetworkConfig)
		if !ok {
			t.Fatalf("expected %T got %T", NetworkConfig{}, desiredState)
		}
		if !reflect.DeepEqual(desiredNetworkConfig, tc.ExpectedNetworkConfig) {
			t.Fatalf("expected %#v got %#v", tc.ExpectedNetworkConfig, desiredNetworkConfig)
		}
	}
}
