package flannel

import (
	"fmt"
	"sync"

	microerror "github.com/giantswarm/microkit/error"
	micrologger "github.com/giantswarm/microkit/logger"
	"github.com/giantswarm/operatorkit/tpr"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/giantswarm/flannel-operator/flag"
	"github.com/giantswarm/flanneltpr"
)

// Config represents the configuration used to create a Crt service.
type Config struct {
	// Dependencies.
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	// Settings.
	Flag  *flag.Flag
	Viper *viper.Viper
}

// DefaultConfig provides a default configuration to create a new create service
// by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		K8sClient: nil,
		Logger:    nil,

		// Settings.
		Flag:  nil,
		Viper: nil,
	}
}

// Service implements the Crt service interface.
type Service struct {
	Config

	// Internals.
	bootOnce sync.Once
	tpr      *tpr.TPR
}

// New creates a new configured Crt service.
func New(config Config) (*Service, error) {
	// Dependencies.
	if config.K8sClient == nil {
		return nil, microerror.MaskAnyf(invalidConfigError, "kubernetes client must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.MaskAnyf(invalidConfigError, "logger must not be empty")
	}

	// Settings.
	if config.Flag == nil {
		return nil, microerror.MaskAnyf(invalidConfigError, "flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.MaskAnyf(invalidConfigError, "viper must not be empty")
	}

	tpr, err := tpr.New(tpr.Config{
		Clientset: config.K8sClient,

		Name:        flanneltpr.Name,
		Version:     flanneltpr.VersionV1,
		Description: flanneltpr.Description,
	})
	if err != nil {
		return nil, microerror.MaskAnyf(err, "creating TPR util")
	}

	newService := &Service{
		Config: config,

		// Internals
		bootOnce: sync.Once{},
		tpr:      tpr,
	}

	return newService, nil
}

// Boot starts the service and implements the watch for the flannel TPR.
func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		err := s.tpr.CreateAndWait()
		switch {
		case tpr.IsAlreadyExists(err):
			s.Logger.Log("info", "flannel third-party resource already exists")
		case err != nil:
			panic(fmt.Sprintf("could not create flannel resource: %#v", err))
		default:
			s.Logger.Log("info", "successfully created flannel third-party resource")
		}
	})
}

// addFunc does nothing as the operator reacts only on TPO delete.
func (s *Service) addFunc(obj interface{}) {}

// deleteFunc waits for the delting cluster's namespace to be fully deleted and
// then cleans up flannel bridges.
func (s *Service) deleteFunc(obj interface{}) {
	panic("TODO: implement")

}

// newListWatch returns a configured list watch for the flannel TPR.
func (s *Service) newListWatch() *cache.ListWatch {
	panic("TODO: implement")
}
