package flannel

import (
	"context"

	"github.com/giantswarm/flanneltpr"
	microerror "github.com/giantswarm/microkit/error"
	"github.com/giantswarm/microkit/storage"
	"github.com/giantswarm/microkit/tls"
)

func (s *Service) cleanupEtcd(spec flanneltpr.Spec) error {
	var (
		endpoint  = s.Viper.GetString(s.Flag.Service.Etcd.Endpoint)
		certFiles = tls.CertFiles{
			RootCAs: []string{s.Viper.GetString(s.Flag.Service.Etcd.TLS.CAFile)},
			Cert:    s.Viper.GetString(s.Flag.Service.Etcd.TLS.CrtFile),
			Key:     s.Viper.GetString(s.Flag.Service.Etcd.TLS.KeyFile),
		}
		path = "coreos.com/network/" + networkBridgeName(spec)
	)

	config := storage.DefaultConfig()
	config.EtcdAddress = endpoint
	config.EtcdTLS = certFiles
	config.Kind = storage.KindEtcdV2

	store, err := storage.New(config)
	if err != nil {
		return microerror.MaskAnyf(err, "creating storage for etcd endpoint %s and certificates %#v", endpoint, certFiles)
	}

	err = store.Delete(context.TODO(), path)
	if err != nil {
		return microerror.MaskAnyf(err, "deleting etcd path %s", path)
	}

	return nil
}
