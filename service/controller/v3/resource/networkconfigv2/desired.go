package networkconfigv2

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	networkConfig := NetworkConfig{
		Network:   customObject.Spec.Flannel.Spec.Network,
		SubnetLen: customObject.Spec.Flannel.Spec.SubnetLen,
		Backend: Backend{
			Type: BackendType,
			VNI:  customObject.Spec.Flannel.Spec.VNI,
		},
	}

	return networkConfig, nil
}
