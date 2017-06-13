package flannel

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/coreos/etcd/pkg/tlsutil"
	"github.com/giantswarm/flanneltpr"
	microerror "github.com/giantswarm/microkit/error"
	"github.com/giantswarm/microkit/storage"
	"github.com/giantswarm/microkit/storage/etcdv2"
)

type certFiles struct {
	CA   []string
	Cert string
	Key  string
}

func (s *Service) cleanupEtcd(spec flanneltpr.Spec) error {
	var (
		endpoint  = s.Viper.GetString(s.Flag.Service.Etcd.Endpoint)
		certFiles = certFiles{
			CA:   []string{s.Viper.GetString(s.Flag.Service.Etcd.TLS.CAFile)},
			Cert: s.Viper.GetString(s.Flag.Service.Etcd.TLS.CrtFile),
			Key:  s.Viper.GetString(s.Flag.Service.Etcd.TLS.KeyFile),
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
	etcdClient, err := newEtcdClient(endpoint, certFiles)
	if err != nil {
		return nil, microerror.MaskAnyf(err, "creating etcd client")
	}

	config := etcdv2.Config{
		EtcdClient: etcdClient,
	}

	store, err := etcdv2.New(config)
	if err != nil {
		return nil, microerror.MaskAny(err)
	}
	return store, nil
}

func newEtcdClient(endpoint string, certFiles certFiles) (client.Client, error) {
	tlsConfig, err := loadTLSConfig(certFiles)
	if err != nil {
		return nil, microerror.MaskAny(err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
	}

	config := client.Config{
		Endpoints:               []string{endpoint},
		Transport:               transport,
		HeaderTimeoutPerRequest: time.Second * 5,
	}

	client, err := client.New(config)
	if err != nil {
		return nil, microerror.MaskAny(err)
	}
	return client, nil
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
