package networkconfigv1

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/flannel-operator/service/keyv1"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	customObject, err := keyv1.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	networkConfigToCreate, err := toNetworkConfig(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	var emptyNetworkConfig NetworkConfig
	if networkConfigToCreate != emptyNetworkConfig {
		b, err := json.Marshal(networkConfigToCreate)
		if err != nil {
			return microerror.Mask(err)
		}
		p := keyv1.EtcdNetworkConfigPath(customObject)
		err = r.store.Create(ctx, p, string(b))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentNetworkConfig, err := toNetworkConfig(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredNetworkConfig, err := toNetworkConfig(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var networkConfigToCreate NetworkConfig
	{
		var emptyNetworkConfig NetworkConfig
		if currentNetworkConfig == emptyNetworkConfig {
			networkConfigToCreate = desiredNetworkConfig
		}
	}

	return networkConfigToCreate, nil
}
