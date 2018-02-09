package networkconfigv2

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/flannel-operator/service/flannelconfig/v2/keyv2"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := keyv2.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var networkConfig NetworkConfig
	{
		p := keyv2.EtcdNetworkConfigPath(customObject)
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
