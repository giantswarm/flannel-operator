package networkconfig

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/flannel-operator/service/key"
)

func (r *Resource) GetDeleteState(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
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

func (r *Resource) ProcessDeleteState(ctx context.Context, obj, deleteState interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	networkConfigToDelete, err := toNetworkConfig(deleteState)
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
