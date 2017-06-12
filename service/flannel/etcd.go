package flannel

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/tlsutil"
	"github.com/giantswarm/flanneltpr"
	microerror "github.com/giantswarm/microkit/error"
	"github.com/giantswarm/microkit/storage"
	"github.com/giantswarm/microkit/storage/etcd"
)

type certFiles struct {
	Cert string
	Key  string
	CA   []string
}

func (s *Service) cleanupEtcd(spec flanneltpr.Spec) error {
	var (
		endpoint  = s.Viper.GetString(s.Flag.Service.Etcd.Endpoint)
		certFiles = certFiles{
			Cert: "/etc/kubernetes/ssl/etcd/etcd.pem",
			Key:  "/etc/kubernetes/ssl/etcd/etcd-key.pem",
			CA:   []string{"/etc/kubernetes/ssl/etcd/etcd-ca.pem"},
		}
		path = "coreos.com/network/" + networkBridgeName(spec)
	)

	store, err := newStorage(endpoint, certFiles)
	if err != nil {
		return microerror.MaskAnyf(err, "creating storage for etcd endpoint %s and certificates %#v", endpoint, certFiles)
	}

	err = store.Delete(context.TODO(), path)
	if err != nil {
		return microerror.MaskAnyf(err, "deleting etcd path %s", path)
	}

	return nil
}

func newStorage(endpoint string, certFiles certFiles) (storage.Service, error) {
	tlsConfig, err := loadTLSConfig(certFiles)
	if err != nil {
		return nil, microerror.MaskAny(err)
	}

	etcdConfig := clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 5 * time.Second,
		TLS:         tlsConfig,
	}
	etcdClient, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, microerror.MaskAny(err)
	}

	config := etcd.DefaultConfig()
	config.EtcdClient = etcdClient
	store, err := etcd.New(config)
	return store, microerror.MaskAny(err)
}

func loadTLSConfig(files certFiles) (*tls.Config, error) {
	tlsCert, err := tlsutil.NewCert(files.Cert, files.Key, nil)
	if err != nil {
		return nil, microerror.MaskAny(err)
	}

	caCertPool, err := tlsutil.NewCertPool(files.CA)
	if err != nil {
		return nil, microerror.MaskAny(err)
	}

	tlsConfig := tls.Config{
		Certificates: []tls.Certificate{*tlsCert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}
	return &tlsConfig, nil
}
