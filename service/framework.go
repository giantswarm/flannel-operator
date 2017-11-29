package service

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/coreos/etcd/client"
	"github.com/giantswarm/flanneltpr"
	"github.com/giantswarm/microerror"
	microtls "github.com/giantswarm/microkit/tls"
	"github.com/giantswarm/operatorkit/client/k8sclient"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/giantswarm/operatorkit/framework/resource/metricsresource"
	"github.com/giantswarm/operatorkit/framework/resource/retryresource"
	"github.com/giantswarm/operatorkit/informer"
	"github.com/giantswarm/operatorkit/tpr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/flannel-operator/service/etcdv2"
	"github.com/giantswarm/flannel-operator/service/resource/legacyv1"
	"github.com/giantswarm/flannel-operator/service/resource/namespacev1"
	"github.com/giantswarm/flannel-operator/service/resource/networkconfigv1"
)

const (
	ResourceRetries uint64 = 3
)

func newCustomObjectFramework(config Config) (*framework.Framework, error) {
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Viper must not be empty")
	}

	var err error

	var k8sClient kubernetes.Interface
	{
		k8sConfig := k8sclient.DefaultConfig()

		k8sConfig.Logger = config.Logger

		k8sConfig.Address = config.Viper.GetString(config.Flag.Service.Kubernetes.Address)
		k8sConfig.InCluster = config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster)
		k8sConfig.TLS.CAFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile)
		k8sConfig.TLS.CrtFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile)
		k8sConfig.TLS.KeyFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile)

		k8sClient, err = k8sclient.New(k8sConfig)
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
		legacyConfig := legacyv1.DefaultConfig()

		legacyConfig.BackOff = legacyResourceBackOff
		legacyConfig.K8sClient = k8sClient
		legacyConfig.Logger = config.Logger

		legacyConfig.EtcdCAFile = config.Viper.GetString(config.Flag.Service.Etcd.TLS.CAFile)
		legacyConfig.EtcdCrtFile = config.Viper.GetString(config.Flag.Service.Etcd.TLS.CrtFile)
		legacyConfig.EtcdKeyFile = config.Viper.GetString(config.Flag.Service.Etcd.TLS.KeyFile)

		legacyResource, err = legacyv1.New(legacyConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var networkConfigResource framework.Resource
	{
		c := networkconfigv1.DefaultConfig()

		c.Logger = config.Logger
		c.Store = storageService

		networkConfigResource, err = networkconfigv1.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceResource framework.Resource
	{
		namespaceConfig := namespacev1.DefaultConfig()

		namespaceConfig.K8sClient = k8sClient
		namespaceConfig.Logger = config.Logger

		namespaceResource, err = namespacev1.New(namespaceConfig)
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

	initCtxFunc := func(ctx context.Context, obj interface{}) (context.Context, error) {
		return ctx, nil
	}

	var newTPR *tpr.TPR
	{
		c := tpr.DefaultConfig()

		c.K8sClient = k8sClient
		c.Logger = config.Logger

		c.Description = flanneltpr.Description
		c.Name = flanneltpr.Name
		c.Version = flanneltpr.VersionV1

		newTPR, err = tpr.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var newWatcherFactory informer.WatcherFactory
	{
		zeroObjectFactory := &informer.ZeroObjectFactoryFuncs{
			NewObjectFunc:     func() runtime.Object { return &flanneltpr.CustomObject{} },
			NewObjectListFunc: func() runtime.Object { return &flanneltpr.List{} },
		}
		newWatcherFactory = informer.NewWatcherFactory(k8sClient.Discovery().RESTClient(), newTPR.WatchEndpoint(""), zeroObjectFactory)
	}

	var newInformer *informer.Informer
	{
		informerConfig := informer.DefaultConfig()

		informerConfig.WatcherFactory = newWatcherFactory

		informerConfig.RateWait = time.Second * 10
		informerConfig.ResyncPeriod = time.Minute * 5

		newInformer, err = informer.New(informerConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var customObjectFramework *framework.Framework
	{
		c := framework.DefaultConfig()

		c.Informer = newInformer
		c.InitCtxFunc = initCtxFunc
		c.Logger = config.Logger
		c.ResourceRouter = framework.DefaultResourceRouter(resources)
		c.TPR = newTPR

		customObjectFramework, err = framework.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return customObjectFramework, nil
}
