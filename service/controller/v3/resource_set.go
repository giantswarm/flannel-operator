package v3

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	microtls "github.com/giantswarm/microkit/tls"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/crud"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"

	"github.com/giantswarm/flannel-operator/service/controller/v3/etcd"
	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
	"github.com/giantswarm/flannel-operator/service/controller/v3/resource/clusterrolebindings"
	"github.com/giantswarm/flannel-operator/service/controller/v3/resource/flanneld"
	"github.com/giantswarm/flannel-operator/service/controller/v3/resource/legacy"
	"github.com/giantswarm/flannel-operator/service/controller/v3/resource/namespace"
	"github.com/giantswarm/flannel-operator/service/controller/v3/resource/networkconfig"
)

type ResourceSetConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CAFile        string
	CrtFile       string
	EtcdEndpoints []string
	KeyFile       string
}

func NewResourceSet(config ResourceSetConfig) (*controller.ResourceSet, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sClient must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	if config.CrtFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.CrtFile must not be empty")
	}
	if len(config.EtcdEndpoints) == 0 {
		return nil, microerror.Maskf(invalidConfigError, "config.EtcdEndpoints must not be empty")
	}
	if config.KeyFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.KeyFile must not be empty")
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
			Endpoints: config.EtcdEndpoints,
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

	var clusterRoleBindingsResource resource.Interface
	{
		c := clusterrolebindings.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		clusterRoleBindingsResource, err = clusterrolebindings.NewResource(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var flanneldResource resource.Interface
	{
		c := flanneld.Config{
			EtcdEndpoints: config.EtcdEndpoints,
			K8sClient:     config.K8sClient.K8sClient(),
			Logger:        config.Logger,

			EtcdCAFile:  config.CAFile,
			EtcdCrtFile: config.CrtFile,
			EtcdKeyFile: config.KeyFile,
		}

		ops, err := flanneld.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		flanneldResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var legacyResource resource.Interface
	{
		legacyConfig := legacy.DefaultConfig()

		legacyConfig.BackOff = backoff.NewExponential(5*time.Minute, 1*time.Minute)
		legacyConfig.K8sClient = config.K8sClient.K8sClient()
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

	var networkConfigResource resource.Interface
	{
		c := networkconfig.Config{
			Logger: config.Logger,
			Store:  storageService,
		}

		ops, err := networkconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		networkConfigResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceResource resource.Interface
	{
		c := namespace.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		ops, err := namespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		namespaceResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		clusterRoleBindingsResource,
		networkConfigResource,
		namespaceResource,
		legacyResource,
		flanneldResource,
	}

	{
		c := retryresource.WrapConfig{
			Logger: config.Logger,
		}

		resources, err = retryresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		c := metricsresource.WrapConfig{}

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

		return false
	}

	var resourceSet *controller.ResourceSet
	{
		c := controller.ResourceSetConfig{
			Handles:   handlesFunc,
			Logger:    config.Logger,
			Resources: resources,
		}

		resourceSet, err = controller.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resourceSet, nil
}

func toCRUDResource(logger micrologger.Logger, ops crud.Interface) (resource.Interface, error) {
	c := crud.ResourceConfig{
		CRUD:   ops,
		Logger: logger,
	}

	r, err := crud.NewResource(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r, nil
}
