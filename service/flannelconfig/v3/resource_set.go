package v3

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/coreos/etcd/client"
	"github.com/giantswarm/microerror"
	microtls "github.com/giantswarm/microkit/tls"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/giantswarm/operatorkit/framework/resource/metricsresource"
	"github.com/giantswarm/operatorkit/framework/resource/retryresource"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/flannel-operator/service/flannelconfig/v3/etcd"
	"github.com/giantswarm/flannel-operator/service/flannelconfig/v3/key"
	"github.com/giantswarm/flannel-operator/service/flannelconfig/v3/resource/legacy"
	"github.com/giantswarm/flannel-operator/service/flannelconfig/v3/resource/namespace"
	"github.com/giantswarm/flannel-operator/service/flannelconfig/v3/resource/networkconfigv2"
)

const (
	ResourceRetries uint64 = 3
)

type ResourceSetConfig struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	CAFile       string
	CrtFile      string
	EtcdEndpoint string
	KeyFile      string
	ProjectName  string
}

func NewResourceSet(config ResourceSetConfig) (*framework.ResourceSet, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sClient must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	if config.CrtFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.CrtFile must not be empty")
	}
	if config.EtcdEndpoint == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.EtcdEndpoint must not be empty")
	}
	if config.KeyFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.KeyFile must not be empty")
	}
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.ProjectName must not be empty")
	}

	var err error

	var tlsConfig *tls.Config
	{
		rootCAs := []string{}
		if config.CAFile != "" {
			rootCAs = []string{
				config.CAFile,
			}
		}
		certFiles := microtls.CertFiles{
			RootCAs: rootCAs,
			Cert:    config.CrtFile,
			Key:     config.KeyFile,
		}

		tlsConfig, err = microtls.LoadTLSConfig(certFiles)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var storageService *etcd.Service
	{
		etcdConfig := client.Config{
			Endpoints: []string{
				config.EtcdEndpoint,
			},
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				TLSHandshakeTimeout: 10 * time.Second,
				TLSClientConfig:     tlsConfig,
			},
		}
		etcdClient, err := client.New(etcdConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		storageConfig := etcd.DefaultConfig()
		storageConfig.EtcdClient = etcdClient
		storageService, err = etcd.New(storageConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var legacyResourceBackOff *backoff.ExponentialBackOff
	{
		legacyResourceBackOff = backoff.NewExponentialBackOff()
		legacyResourceBackOff.MaxElapsedTime = 5 * time.Minute
	}

	var legacyResource framework.Resource
	{
		legacyConfig := legacy.DefaultConfig()

		legacyConfig.BackOff = legacyResourceBackOff
		legacyConfig.K8sClient = config.K8sClient
		legacyConfig.Logger = config.Logger

		legacyConfig.EtcdCAFile = config.CAFile
		legacyConfig.EtcdCrtFile = config.CrtFile
		legacyConfig.EtcdKeyFile = config.KeyFile

		ops, err := legacy.New(legacyConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		legacyResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var networkConfigResource framework.Resource
	{
		c := networkconfigv2.DefaultConfig()

		c.Logger = config.Logger
		c.Store = storageService

		ops, err := networkconfigv2.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		networkConfigResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceResource framework.Resource
	{
		c := namespace.DefaultConfig()

		c.K8sClient = config.K8sClient
		c.Logger = config.Logger

		ops, err := namespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		namespaceResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []framework.Resource{
		networkConfigResource,
		namespaceResource,
		legacyResource,
	}

	{
		c := retryresource.WrapConfig{
			BackOffFactory: func() backoff.BackOff { return backoff.WithMaxTries(backoff.NewExponentialBackOff(), ResourceRetries) },
			Logger:         config.Logger,
		}

		resources, err = retryresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		c := metricsresource.WrapConfig{
			Name: config.ProjectName,
		}

		resources, err = metricsresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	handlesFunc := func(obj interface{}) bool {
		customObject, err := key.ToCustomObject(obj)
		if err != nil {
			return false
		}

		if key.VersionBundleVersion(customObject) == VersionBundle().Version {
			return true
		}
		// TODO remove this hack with the next version bundle version or as soon as
		// all certconfigs obtain a real version bundle version.
		if key.VersionBundleVersion(customObject) == "" {
			return true
		}

		return false
	}

	var resourceSet *framework.ResourceSet
	{
		c := framework.ResourceSetConfig{
			Handles:   handlesFunc,
			Logger:    config.Logger,
			Resources: resources,
		}

		resourceSet, err = framework.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resourceSet, nil
}

func toCRUDResource(logger micrologger.Logger, ops framework.CRUDResourceOps) (*framework.CRUDResource, error) {
	c := framework.CRUDResourceConfig{
		Logger: logger,
		Ops:    ops,
	}

	r, err := framework.NewCRUDResource(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r, nil
}
