package networkconfig

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/flannel-operator/service/key"
)

func (r *Resource) GetUpdateState(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, interface{}, interface{}, error) {
	currentNetworkConfig, err := toNetworkConfig(currentState)
	if err != nil {
		return nil, nil, nil, microerror.Mask(err)
	}
	desiredNetworkConfig, err := toNetworkConfig(desiredState)
	if err != nil {
		return nil, nil, nil, microerror.Mask(err)
	}

	// We do not compute the create state here because the framework already
	// handles it separately.
	var networkConfigToCreate NetworkConfig

	// We do not compute the delete state here because there are no resources we
	// can safely remove.
	var networkConfigToDelete NetworkConfig

	var networkConfigToUpdate NetworkConfig
	if currentNetworkConfig != desiredNetworkConfig {
		networkConfigToUpdate = desiredNetworkConfig
	}

	return networkConfigToCreate, networkConfigToDelete, networkConfigToUpdate, nil
}

func (r *Resource) ProcessUpdateState(ctx context.Context, obj, updateState interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	networkConfigToUpdate, err := toNetworkConfig(updateState)
	if err != nil {
		return microerror.Mask(err)
	}

	var emptyNetworkConfig NetworkConfig
	if networkConfigToUpdate != emptyNetworkConfig {
		p := key.EtcdNetworkPath(customObject)
		err = r.store.Delete(ctx, p)
		if err != nil {
			return microerror.Mask(err)
		}

		b, err := json.Marshal(networkConfigToUpdate)
		if err != nil {
			return microerror.Mask(err)
		}
		p = key.EtcdNetworkConfigPath(customObject)
		err = r.store.Create(ctx, p, string(b))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
