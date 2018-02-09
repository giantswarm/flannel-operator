package networkconfigv2

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/giantswarm/flannel-operator/service/flannelconfig/v2/keyv2"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	customObject, err := keyv2.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	networkConfigToUpdate, err := toNetworkConfig(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	var emptyNetworkConfig NetworkConfig
	if networkConfigToUpdate != emptyNetworkConfig {
		p := keyv2.EtcdNetworkPath(customObject)
		err = r.store.Delete(ctx, p)
		if err != nil {
			return microerror.Mask(err)
		}

		b, err := json.Marshal(networkConfigToUpdate)
		if err != nil {
			return microerror.Mask(err)
		}
		p = keyv2.EtcdNetworkConfigPath(customObject)
		err = r.store.Create(ctx, p, string(b))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	create, err := r.newCreateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := framework.NewPatch()
	patch.SetCreateChange(create)
	patch.SetUpdateChange(update)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentNetworkConfig, err := toNetworkConfig(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredNetworkConfig, err := toNetworkConfig(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var networkConfigToUpdate NetworkConfig
	if currentNetworkConfig != desiredNetworkConfig {
		networkConfigToUpdate = desiredNetworkConfig
	}

	return networkConfigToUpdate, nil
}
