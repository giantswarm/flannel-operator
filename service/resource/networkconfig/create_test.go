package networkconfig

import (
	"context"
	"reflect"
	"testing"

	"github.com/giantswarm/flanneltpr"
	"github.com/giantswarm/micrologger/microloggertest"

	etcdv2fake "github.com/giantswarm/flannel-operator/service/etcdv2/fake"
)

func Test_Resource_NetworkConfig_newCreateChange(t *testing.T) {
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

		// Test 1 ensures that the create state matches the desired state in case
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

		// Test 2 ensures that the create state is empty in case the current state
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

		// Test 3 ensures that the create state is empty in case the current state
		// is not empty. NOTE that this is like test 2 but with incomplete or wrong
		// configuration. The creation process only takes care about configuration
		// existance. The update process takes care about keeping the existing
		// configuration in the right shape.
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
			ExpectedNetworkConfig: NetworkConfig{},
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
		createState, err := newResource.newCreateChange(context.TODO(), tc.Obj, tc.CurrentState, tc.DesiredState)
		if err != nil {
			t.Fatalf("expected %#v got %#v", nil, err)
		}

		t.Run("ValidateCreateState", func(t *testing.T) {
			networkConfigToCreate, ok := createState.(NetworkConfig)
			if !ok {
				t.Fatalf("expected %T got %T", NetworkConfig{}, createState)
			}
			if !reflect.DeepEqual(networkConfigToCreate, tc.ExpectedNetworkConfig) {
				t.Fatalf("expected %#v got %#v", tc.ExpectedNetworkConfig, networkConfigToCreate)
			}
		})
	}
}
