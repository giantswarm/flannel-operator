// Package service implements business logic to issue flannelificates for clusters
// running on the Giantnetes platform.
package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/cenk/backoff"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8s"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/flannel-operator/flag"
	"github.com/giantswarm/flannel-operator/service/flannel"
	"github.com/giantswarm/flannel-operator/service/healthz"
)

// Config represents the configuration used to create a new service.
type Config struct {
	// Dependencies.
	Logger      micrologger.Logger
	VaultClient *vaultapi.Client

	// Settings.
	Flag  *flag.Flag
	Viper *viper.Viper

	Description string
	GitCommit   string
	Name        string
	Source      string
}

// DefaultConfig provides a default configuration to create a new service by
// best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		Logger:      nil,
		VaultClient: nil,

		// Settings.
		Flag:  nil,
		Viper: nil,

		Description: "",
		GitCommit:   "",
		Name:        "",
		Source:      "",
	}
}

// New creates a new configured service object.
func New(config Config) (*Service, error) {
	// Dependencies.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "flag must be set")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "logger must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "viper must be set")
	}

	config.Logger.Log("debug", fmt.Sprintf("creating flannel-operator with config: %#v", config))

	var err error

	var k8sClient kubernetes.Interface
	{
		c := k8s.Config{
			Logger: config.Logger,
			TLS: k8s.TLSClientConfig{
				CAFile:  config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile),
				CrtFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile),
				KeyFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile),
			},
			Address:   config.Viper.GetString(config.Flag.Service.Kubernetes.Address),
			InCluster: config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster),
		}
		k8sClient, err = k8s.NewClient(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorBackOff *backoff.ExponentialBackOff
	{
		operatorBackOff = backoff.NewExponentialBackOff()
		operatorBackOff.MaxElapsedTime = 5 * time.Minute
	}

	var flannelService *flannel.Service
	{
		c := flannel.DefaultConfig()

		c.BackOff = operatorBackOff
		c.K8sClient = k8sClient
		c.Logger = config.Logger

		c.Flag = config.Flag
		c.Viper = config.Viper

		flannelService, err = flannel.New(c)
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
		c := version.DefaultConfig()

		c.Description = config.Description
		c.GitCommit = config.GitCommit
		c.Name = config.Name
		c.Source = config.Source

		versionService, err = version.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	newService := &Service{
		// Dependencies.
		Flannel: flannelService,
		Healthz: healthzService,
		Version: versionService,

		// Internals
		bootOnce: sync.Once{},
	}

	return newService, nil
}

type Service struct {
	// Dependencies.
	Flannel *flannel.Service
	Healthz *healthz.Service
	Version *version.Service

	// Internals.
	bootOnce sync.Once
}

func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		s.Flannel.Boot()
	})
}
