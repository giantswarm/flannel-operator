package service

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/coreos/etcd/client"
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/flanneltpr"
	"github.com/giantswarm/microerror"
	microtls "github.com/giantswarm/microkit/tls"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/giantswarm/operatorkit/client/k8sclient"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/client/k8sextclient"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/giantswarm/operatorkit/framework/resource/metricsresource"
	"github.com/giantswarm/operatorkit/framework/resource/retryresource"
	"github.com/giantswarm/operatorkit/informer"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/flannel-operator/service/etcdv2"
	"github.com/giantswarm/flannel-operator/service/resource/legacyv2"
	"github.com/giantswarm/flannel-operator/service/resource/namespacev2"
	"github.com/giantswarm/flannel-operator/service/resource/networkconfigv2"
)

const (
	ResourceRetries uint64 = 3
)

const (
	FlannelConfigCleanupFinalizer = "flannel-operator.giantswarm.io/custom-object-cleanup"
)

func newCRDFramework(config Config) (*framework.Framework, error) {
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Viper must not be empty")
	}

	var err error

	var k8sClient kubernetes.Interface
	{
		c := k8sclient.DefaultConfig()

		c.Logger = config.Logger

		c.Address = config.Viper.GetString(config.Flag.Service.Kubernetes.Address)
		c.InCluster = config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster)
		c.TLS.CAFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile)
		c.TLS.CrtFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile)
		c.TLS.KeyFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile)

		k8sClient, err = k8sclient.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var k8sExtClient apiextensionsclient.Interface
	{
		c := k8sextclient.DefaultConfig()

		c.Logger = config.Logger

		c.Address = config.Viper.GetString(config.Flag.Service.Kubernetes.Address)
		c.InCluster = config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster)
		c.TLS.CAFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile)
		c.TLS.CrtFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile)
		c.TLS.KeyFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile)

		k8sExtClient, err = k8sextclient.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var crdClient *k8scrdclient.CRDClient
	{
		c := k8scrdclient.DefaultConfig()

		c.K8sExtClient = k8sExtClient
		c.Logger = microloggertest.New()

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
		legacyConfig := legacyv2.DefaultConfig()

		legacyConfig.BackOff = legacyResourceBackOff
		legacyConfig.K8sClient = k8sClient
		legacyConfig.Logger = config.Logger

		legacyConfig.EtcdCAFile = config.Viper.GetString(config.Flag.Service.Etcd.TLS.CAFile)
		legacyConfig.EtcdCrtFile = config.Viper.GetString(config.Flag.Service.Etcd.TLS.CrtFile)
		legacyConfig.EtcdKeyFile = config.Viper.GetString(config.Flag.Service.Etcd.TLS.KeyFile)

		legacyResource, err = legacyv2.New(legacyConfig)
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

	var clientSet *versioned.Clientset
	{
		var c *rest.Config

		if config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster) {
			config.Logger.Log("debug", "creating in-cluster config")

			c, err = rest.InClusterConfig()
			if err != nil {
				return nil, microerror.Mask(err)
			}
		} else {
			config.Logger.Log("debug", "creating out-cluster config")

			c = &rest.Config{
				Host: config.Viper.GetString(config.Flag.Service.Kubernetes.Address),
				TLSClientConfig: rest.TLSClientConfig{
					CAFile:   config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile),
					CertFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile),
					KeyFile:  config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile),
				},
			}
		}

		clientSet, err = versioned.NewForConfig(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// TODO remove after migration.
	migrateTPRsToCRDs(config.Logger, clientSet)

	var newWatcherFactory informer.WatcherFactory
	{
		newWatcherFactory = func() (watch.Interface, error) {
			watcher, err := clientSet.CoreV1alpha1().FlannelConfigs("").Watch(apismetav1.ListOptions{})
			if err != nil {
				return nil, microerror.Mask(err)
			}

			return watcher, nil
		}
	}

	var newInformer *informer.Informer
	{
		c := informer.DefaultConfig()

		c.WatcherFactory = newWatcherFactory

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

func migrateTPRsToCRDs(logger micrologger.Logger, clientSet *versioned.Clientset) {
	logger.Log("debug", "start TPR migration")

	var err error

	// List all TPOs.
	var b []byte
	{
		e := "apis/giantswarm.io/v1/namespaces/default/flannelnetworks"
		b, err = clientSet.Discovery().RESTClient().Get().AbsPath(e).DoRaw()
		if err != nil {
			logger.Log("error", fmt.Sprintf("%#v", err))
			return
		}

		fmt.Printf("\n")
		fmt.Printf("b start\n")
		fmt.Printf("%s\n", b)
		fmt.Printf("b end\n")
		fmt.Printf("\n")
	}

	// Convert bytes into structure.
	var v *flanneltpr.List
	{
		v = &flanneltpr.List{}
		if err := json.Unmarshal(b, v); err != nil {
			logger.Log("error", fmt.Sprintf("%#v", err))
			return
		}

		fmt.Printf("\n")
		fmt.Printf("v start\n")
		fmt.Printf("%#v\n", v)
		fmt.Printf("v end\n")
		fmt.Printf("\n")
	}

	// Iterate over all TPOs.
	for _, tpo := range v.Items {
		// Compute CRO using TPO.
		var cro *v1alpha1.FlannelConfig
		{
			cro = &v1alpha1.FlannelConfig{}

			cro.TypeMeta.APIVersion = "core.giantswarm.io"
			cro.TypeMeta.Kind = "FlannelConfig"
			cro.ObjectMeta.Name = tpo.Name
			//cro.ObjectMeta.Finalizers = []string{
			//	FlannelConfigCleanupFinalizer,
			//}
			cro.Spec.Bridge.Docker.Image = tpo.Spec.Bridge.Docker.Image
			cro.Spec.Bridge.Spec.DNS.Servers = toStrings(tpo.Spec.Bridge.Spec.DNS.Servers)
			cro.Spec.Bridge.Spec.Interface = tpo.Spec.Bridge.Spec.Interface
			cro.Spec.Bridge.Spec.NTP.Servers = tpo.Spec.Bridge.Spec.NTP.Servers
			cro.Spec.Bridge.Spec.PrivateNetwork = tpo.Spec.Bridge.Spec.PrivateNetwork
			cro.Spec.Cluster.Customer = tpo.Spec.Cluster.Customer
			cro.Spec.Cluster.ID = tpo.Spec.Cluster.ID
			cro.Spec.Cluster.Namespace = tpo.Spec.Cluster.Namespace
			cro.Spec.Flannel.Spec.Network = tpo.Spec.Flannel.Spec.Network
			cro.Spec.Flannel.Spec.RunDir = tpo.Spec.Flannel.Spec.RunDir
			cro.Spec.Flannel.Spec.SubnetLen = tpo.Spec.Flannel.Spec.SubnetLen
			cro.Spec.Flannel.Spec.VNI = tpo.Spec.Flannel.Spec.VNI
			cro.Spec.Health.Docker.Image = tpo.Spec.Health.Docker.Image
			cro.Spec.VersionBundle.Version = tpo.Spec.VersionBundle.Version

			fmt.Printf("\n")
			fmt.Printf("cro start\n")
			fmt.Printf("%#v\n", cro)
			fmt.Printf("cro end\n")
			fmt.Printf("\n")
		}

		// Create CRO in Kubernetes API.
		{
			_, err := clientSet.CoreV1alpha1().FlannelConfigs(tpo.Namespace).Get(cro.Name, apismetav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				_, err := clientSet.CoreV1alpha1().FlannelConfigs(tpo.Namespace).Create(cro)
				if err != nil {
					logger.Log("error", fmt.Sprintf("%#v", err))
					return
				}
			} else if err != nil {
				logger.Log("error", fmt.Sprintf("%#v", err))
				return
			}
		}
	}

	logger.Log("debug", "end TPR migration")
}

func toStrings(ipList []net.IP) []string {
	var newList []string

	for _, ip := range ipList {
		newList = append(newList, ip.String())
	}

	return newList
}
