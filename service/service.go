// Package service implements business logic to issue flannelificates for clusters
// running on the Giantnetes platform.
package service

import (
	"sync"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/client/k8srestconfig"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/spf13/viper"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/flannel-operator/flag"
	"github.com/giantswarm/flannel-operator/service/flannelconfig"
	"github.com/giantswarm/flannel-operator/service/healthz"
)

// Config represents the configuration used to create a new service.
type Config struct {
	// Dependencies.
	Logger micrologger.Logger

	// Settings.
	Description string
	Flag        *flag.Flag
	GitCommit   string
	Name        string
	Source      string
	Viper       *viper.Viper
}

// DefaultConfig provides a default configuration to create a new service by
// best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		Logger: nil,

		// Settings.
		Description: "",
		Flag:        nil,
		GitCommit:   "",
		Name:        "",
		Source:      "",
		Viper:       nil,
	}
}

type Service struct {
	// Dependencies.
	FlannelConfigFramework *framework.Framework
	Healthz                *healthz.Service
	Version                *version.Service

	// Internals.
	bootOnce sync.Once
}

func New(config Config) (*Service, error) {
	// Settings.
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

	var flannelConfigFramework *framework.Framework
	{
		c := flannelconfig.FrameworkConfig{
			CRDClient: crdClient,
			G8sClient: g8sClient,
			K8sClient: k8sClient,
			Logger:    config.Logger,

			CAFile:       config.Viper.GetString(config.Flag.Service.Etcd.TLS.CAFile),
			CrtFile:      config.Viper.GetString(config.Flag.Service.Etcd.TLS.CrtFile),
			EtcdEndpoint: config.Viper.GetString(config.Flag.Service.Etcd.Endpoint),
			KeyFile:      config.Viper.GetString(config.Flag.Service.Etcd.TLS.KeyFile),
			ProjectName:  config.Name,
		}
		flannelConfigFramework, err = flannelconfig.NewFramework(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var healthzService *healthz.Service
	{
		healthzConfig := healthz.DefaultConfig()

		healthzConfig.K8sClient = k8sClient
		healthzConfig.Logger = config.Logger

		healthzService, err = healthz.New(healthzConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var versionService *version.Service
	{
		versionConfig := version.DefaultConfig()

		versionConfig.Description = config.Description
		versionConfig.GitCommit = config.GitCommit
		versionConfig.Name = config.Name
		versionConfig.Source = config.Source
		versionConfig.VersionBundles = NewVersionBundles()

		versionService, err = version.New(versionConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	newService := &Service{
		// Dependencies.
		FlannelConfigFramework: flannelConfigFramework,
		Healthz:                healthzService,
		Version:                versionService,

		// Internals
		bootOnce: sync.Once{},
	}

	return newService, nil
}

func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		go s.FlannelConfigFramework.Boot()
	})
}
