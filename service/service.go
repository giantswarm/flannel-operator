// Package service implements business logic to issue flannelificates for clusters
// running on the Giantnetes platform.
package service

import (
	"fmt"
	"sync"

	microerror "github.com/giantswarm/microkit/error"
	micrologger "github.com/giantswarm/microkit/logger"
	"github.com/giantswarm/operatorkit/client/k8s"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/flannel-operator/flag"
	"github.com/giantswarm/flannel-operator/service/flannel"
	"github.com/giantswarm/flannel-operator/service/version"
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
		return nil, microerror.MaskAnyf(invalidConfigError, "flag must be set")
	}
	if config.Logger == nil {
		return nil, microerror.MaskAnyf(invalidConfigError, "logger must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.MaskAnyf(invalidConfigError, "viper must be set")
	}

	config.Logger.Log("debug", fmt.Sprintf("creating flannel-operator with config: %#v", config))

	var err error

	var k8sClient kubernetes.Interface
	{
		c := k8s.Config{
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
			return nil, microerror.MaskAny(err)
		}
	}

	var flannelService *flannel.Service
	{
		c := flannel.DefaultConfig()

		c.K8sClient = k8sClient
		c.Logger = config.Logger

		c.Flag = config.Flag
		c.Viper = config.Viper

		flannelService, err = flannel.New(c)
		if err != nil {
			return nil, microerror.MaskAny(err)
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
			return nil, microerror.MaskAny(err)
		}
	}

	newService := &Service{
		// Dependencies.
		Flannel: flannelService,
		Version: versionService,

		// Internals
		bootOnce: sync.Once{},
	}

	return newService, nil
}

type Service struct {
	// Dependencies.
	Flannel *flannel.Service
	Version *version.Service

	// Internals.
	bootOnce sync.Once
}

func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		s.Flannel.Boot()
	})
}
