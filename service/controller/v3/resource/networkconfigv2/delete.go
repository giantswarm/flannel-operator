package networkconfigv2

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/giantswarm/flannel-operator/service/flannelconfig/v3/key"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	networkConfigToDelete, err := toNetworkConfig(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	var emptyNetworkConfig NetworkConfig
	if networkConfigToDelete != emptyNetworkConfig {
		p := key.EtcdNetworkPath(customObject)
		err = r.store.Delete(ctx, p)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	delete, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := framework.NewPatch()
	patch.SetDeleteChange(delete)

	return patch, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentNetworkConfig, err := toNetworkConfig(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredNetworkConfig, err := toNetworkConfig(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var networkConfigToDelete NetworkConfig
	{
		var emptyNetworkConfig NetworkConfig
		if currentNetworkConfig != emptyNetworkConfig {
			networkConfigToDelete = desiredNetworkConfig
		}
	}

	return networkConfigToDelete, nil
}
