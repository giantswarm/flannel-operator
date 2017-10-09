package networkconfig

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/flanneltpr"
	"github.com/giantswarm/flanneltpr/spec"
	"github.com/giantswarm/flanneltpr/spec/flannel"
	"github.com/giantswarm/micrologger/microloggertest"

	etcdv2fake "github.com/giantswarm/flannel-operator/service/etcdv2/fake"
)

func Test_Resource_NetworkConfig_GetDesiredState(t *testing.T) {
	testCases := []struct {
		Obj                   interface{}
		ExpectedNetworkConfig NetworkConfig
	}{
		{
			Obj: &flanneltpr.CustomObject{},
			ExpectedNetworkConfig: NetworkConfig{
				Backend: Backend{
					Type: "vxlan",
				},
			},
		},

		{
			Obj: &flanneltpr.CustomObject{
				Spec: flanneltpr.Spec{
					Flannel: spec.Flannel{
						Spec: flannel.Spec{
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
		resourceConfig.Store = etcdv2fake.New()

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
