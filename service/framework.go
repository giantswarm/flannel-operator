package service

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/coreos/etcd/client"
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	microtls "github.com/giantswarm/microkit/tls"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/client/k8srestconfig"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/giantswarm/operatorkit/framework/resource/metricsresource"
	"github.com/giantswarm/operatorkit/framework/resource/retryresource"
	"github.com/giantswarm/operatorkit/informer"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/flannel-operator/service/flannelconfig/v2/etcdv2"
	"github.com/giantswarm/flannel-operator/service/flannelconfig/v2/resource/legacy"
	"github.com/giantswarm/flannel-operator/service/flannelconfig/v2/resource/namespacev2"
	"github.com/giantswarm/flannel-operator/service/flannelconfig/v2/resource/networkconfigv2"
)

const (
	ResourceRetries uint64 = 3
)

func newCRDFramework(config Config) (*framework.Framework, error) {
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Viper must not be empty")
	}

	var err error

	var restConfig *rest.Config
	{
		c := k8srestconfig.DefaultConfig()

		c.Logger = config.Logger

		c.Address = config.Viper.GetString(config.Flag.Service.Kubernetes.Address)
		c.InCluster = config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster)
		c.TLS.CAFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile)
		c.TLS.CrtFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile)
		c.TLS.KeyFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile)

		restConfig, err = k8srestconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	g8sClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	k8sExtClient, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var crdClient *k8scrdclient.CRDClient
	{
		c := k8scrdclient.DefaultConfig()

		c.K8sExtClient = k8sExtClient
		c.Logger = config.Logger

		crdClient, err = k8scrdclient.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var tlsConfig *tls.Config
	{
		rootCAs := []string{}
		{
			v := config.Viper.GetString(config.Flag.Service.Etcd.TLS.CAFile)
			if v != "" {
				rootCAs = []string{v}
			}
		}
		certFiles := microtls.CertFiles{
			RootCAs: rootCAs,
			Cert:    config.Viper.GetString(config.Flag.Service.Etcd.TLS.CrtFile),
			Key:     config.Viper.GetString(config.Flag.Service.Etcd.TLS.KeyFile),
		}

		tlsConfig, err = microtls.LoadTLSConfig(certFiles)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var storageService *etcdv2.Service
	{
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     tlsConfig,
		}

		etcdConfig := client.Config{
			Endpoints: []string{config.Viper.GetString(config.Flag.Service.Etcd.Endpoint)},
			Transport: transport,
		}
		etcdClient, err := client.New(etcdConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		storageConfig := etcdv2.DefaultConfig()
		storageConfig.EtcdClient = etcdClient
		storageService, err = etcdv2.New(storageConfig)
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
		legacyConfig.K8sClient = k8sClient
		legacyConfig.Logger = config.Logger

		legacyConfig.EtcdCAFile = config.Viper.GetString(config.Flag.Service.Etcd.TLS.CAFile)
		legacyConfig.EtcdCrtFile = config.Viper.GetString(config.Flag.Service.Etcd.TLS.CrtFile)
		legacyConfig.EtcdKeyFile = config.Viper.GetString(config.Flag.Service.Etcd.TLS.KeyFile)

		legacyResource, err = legacy.New(legacyConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var networkConfigResource framework.Resource
	{
		c := networkconfigv2.DefaultConfig()

		c.Logger = config.Logger
		c.Store = storageService

		networkConfigResource, err = networkconfigv2.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceResource framework.Resource
	{
		c := namespacev2.DefaultConfig()

		c.K8sClient = k8sClient
		c.Logger = config.Logger

		namespaceResource, err = namespacev2.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resources []framework.Resource
	{
		resources = []framework.Resource{
			networkConfigResource,
			namespaceResource,
			legacyResource,
		}

		retryWrapConfig := retryresource.DefaultWrapConfig()
		retryWrapConfig.BackOffFactory = func() backoff.BackOff { return backoff.WithMaxTries(backoff.NewExponentialBackOff(), ResourceRetries) }
		retryWrapConfig.Logger = config.Logger
		resources, err = retryresource.Wrap(resources, retryWrapConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		metricsWrapConfig := metricsresource.DefaultWrapConfig()
		metricsWrapConfig.Name = config.Name
		resources, err = metricsresource.Wrap(resources, metricsWrapConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var newInformer *informer.Informer
	{
		c := informer.DefaultConfig()

		c.Watcher = g8sClient.CoreV1alpha1().FlannelConfigs("")

		newInformer, err = informer.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var crdFramework *framework.Framework
	{
		c := framework.DefaultConfig()

		c.CRD = v1alpha1.NewFlannelConfigCRD()
		c.CRDClient = crdClient
		c.Informer = newInformer
		c.Logger = config.Logger
		c.ResourceRouter = framework.DefaultResourceRouter(resources)

		crdFramework, err = framework.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return crdFramework, nil
}
