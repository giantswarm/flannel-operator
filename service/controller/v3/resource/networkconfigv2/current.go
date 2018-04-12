package networkconfigv2

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var networkConfig NetworkConfig
	{
		p := key.EtcdNetworkConfigPath(customObject)
		exists, err := r.store.Exists(ctx, p)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if exists {
			s, err := r.store.Search(ctx, p)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			err = json.Unmarshal([]byte(s), &networkConfig)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		}
	}

	return networkConfig, nil
}
