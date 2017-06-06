package flannel

import (
	"context"

	"github.com/giantswarm/flanneltpr"
	microerror "github.com/giantswarm/microkit/error"
	"github.com/giantswarm/microkit/storage"
)

func (s *Service) cleanupEtcd(spec flanneltpr.Spec) error {
	endpoint := s.Viper.GetString(s.Flag.Service.Etcd.Endpoint)
	path := "coreos.com/network/" + networkBridgeName(spec)

	config := storage.Config{
		EtcdAddress: endpoint,
	}

	store, err := storage.New(config)
	if err != nil {
		return microerror.MaskAnyf(err, "creating storage for etcd endpoint %s", endpoint)
	}

	err = store.Delete(context.TODO(), path)
	if err != nil {
		return microerror.MaskAnyf(err, "deleting etcd path %s", path)
	}

	return nil
}
