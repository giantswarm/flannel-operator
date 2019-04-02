package networkconfig

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/flannel-operator/service/controller/v3/etcd"
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
		s, err := r.store.Search(ctx, p)
		if etcd.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			err := json.Unmarshal([]byte(s), &networkConfig)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		}
	}

	return networkConfig, nil
}
