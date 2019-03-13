// Package service implements business logic to issue flannelificates for clusters
// running on the Giantnetes platform.
package service

import (
	"context"
	"sync"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/client/k8srestconfig"
	"github.com/spf13/viper"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/flannel-operator/flag"
	"github.com/giantswarm/flannel-operator/service/controller"
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
	Version *version.Service

	bootOnce          sync.Once
	networkController *controller.Network
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
		c := k8srestconfig.Config{
			Logger: config.Logger,

			Address:   config.Viper.GetString(config.Flag.Service.Kubernetes.Address),
			InCluster: config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster),
			TLS: k8srestconfig.ConfigTLS{
				CAFile:  config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile),
				CrtFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile),
				KeyFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile),
			},
		}

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
		c := k8scrdclient.Config{
			K8sExtClient: k8sExtClient,
			Logger:       config.Logger,
		}

		crdClient, err = k8scrdclient.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var networkController *controller.Network
	{
		c := controller.NetworkConfig{
			CRDClient: crdClient,
			G8sClient: g8sClient,
			K8sClient: k8sClient,
			Logger:    config.Logger,

			CAFile:           config.Viper.GetString(config.Flag.Service.Etcd.TLS.CAFile),
			CrtFile:          config.Viper.GetString(config.Flag.Service.Etcd.TLS.CrtFile),
			CRDLabelSelector: config.Viper.GetString(config.Flag.Service.CRD.LabelSelector),
			EtcdEndpoint:     config.Viper.GetString(config.Flag.Service.Etcd.Endpoint),
			KeyFile:          config.Viper.GetString(config.Flag.Service.Etcd.TLS.KeyFile),
			ProjectName:      config.Name,
		}

		networkController, err = controller.NewNetwork(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var versionService *version.Service
	{
		c := version.Config{
			Description:    config.Description,
			GitCommit:      config.GitCommit,
			Name:           config.Name,
			Source:         config.Source,
			VersionBundles: NewVersionBundles(),
		}

		versionService, err = version.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Service{
		Version: versionService,

		bootOnce:          sync.Once{},
		networkController: networkController,
	}

	return s, nil
}

func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		go s.networkController.Boot(context.Background())
	})
}
