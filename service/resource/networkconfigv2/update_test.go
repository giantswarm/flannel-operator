package networkconfigv1

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/flanneltpr"
	"github.com/giantswarm/micrologger/microloggertest"

	etcdv2fake "github.com/giantswarm/flannel-operator/service/etcdv2/fake"
)

func Test_Resource_NetworkConfig_newUpdateChange(t *testing.T) {
	testCases := []struct {
		Obj                   interface{}
		CurrentState          interface{}
		DesiredState          interface{}
		ExpectedNetworkConfig NetworkConfig
	}{
		// Test 0 ensures that zero value inputs result in zero value outputs.
		{
			Obj:                   &flanneltpr.CustomObject{},
			CurrentState:          NetworkConfig{},
			DesiredState:          NetworkConfig{},
			ExpectedNetworkConfig: NetworkConfig{},
		},

		// Test 1 ensures that the update state matches the desired state in case
		// the current state is empty.
		{
			Obj:          &flanneltpr.CustomObject{},
			CurrentState: NetworkConfig{},
			DesiredState: NetworkConfig{
				Network:   "172.26.0.0/16",
				SubnetLen: 30,
				Backend: Backend{
					Type: "vxlan",
					VNI:  26,
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

		// Test 2 ensures that the update state is empty in case the current state
		// is not empty.
		{
			Obj: &flanneltpr.CustomObject{},
			CurrentState: NetworkConfig{
				Network:   "172.26.0.0/16",
				SubnetLen: 30,
				Backend: Backend{
					Type: "vxlan",
					VNI:  26,
				},
			},
			DesiredState: NetworkConfig{
				Network:   "172.26.0.0/16",
				SubnetLen: 30,
				Backend: Backend{
					Type: "vxlan",
					VNI:  26,
				},
			},
			ExpectedNetworkConfig: NetworkConfig{},
		},

		// Test 3 ensures that the update state matches the desired state in case
		// the current state does not match the desired state.
		{
			Obj: &flanneltpr.CustomObject{},
			CurrentState: NetworkConfig{
				Backend: Backend{
					Type: "vxlan",
				},
			},
			DesiredState: NetworkConfig{
				Network:   "172.26.0.0/16",
				SubnetLen: 30,
				Backend: Backend{
					Type: "vxlan",
					VNI:  26,
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

		// Test 3 is the same as 2 but with different changes between current and
		// desired state.
		{
			Obj: &flanneltpr.CustomObject{},
			CurrentState: NetworkConfig{
				Network:   "foo/16",
				SubnetLen: 30000000000,
				Backend: Backend{
					Type: "vxlan",
					VNI:  0,
				},
			},
			DesiredState: NetworkConfig{
				Network:   "172.26.0.0/16",
				SubnetLen: 30,
				Backend: Backend{
					Type: "vxlan",
					VNI:  26,
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
		updateState, err := newResource.newUpdateChange(context.TODO(), tc.Obj, tc.CurrentState, tc.DesiredState)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		t.Run("ValidateUpdateState", func(t *testing.T) {
			networkConfigToUpdate, ok := updateState.(NetworkConfig)
			if !ok {
				t.Fatalf("expected %T got %T", NetworkConfig{}, updateState)
			}
			if !reflect.DeepEqual(networkConfigToUpdate, tc.ExpectedNetworkConfig) {
				t.Fatalf("expected %#v got %#v", tc.ExpectedNetworkConfig, networkConfigToUpdate)
			}
		})
	}
}
