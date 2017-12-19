package flanneldv1

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"
	"k8s.io/client-go/kubernetes"
)

const (
	// Name is the identifier of the resource.
	Name = "flanneldv1"
)

// Config represents the configuration used to create a new cloud config
// resource.
type Config struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	EtcdCAFile  string
	EtcdCrtFile string
	EtcdKeyFile string
}

// DefaultConfig provides a default configuration to create a new cloud config
// resource by best effort.
func DefaultConfig() Config {
	return Config{
		K8sClient: nil,
		Logger:    nil,

		EtcdCAFile:  "",
		EtcdCrtFile: "",
		EtcdKeyFile: "",
	}
}

// Resource implements the cloud config resource.
type Resource struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	etcdCAFile  string
	etcdCrtFile string
	etcdKeyFile string
}

// New creates a new configured cloud config resource.
func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sClient must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	if config.EtcdCAFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.EtcdCAFile must not be empty")
	}
	if config.EtcdCrtFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.EtcdCrtFile must not be empty")
	}
	if config.EtcdKeyFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.EtcdKeyFile must not be empty")
	}

	r := &Resource{
		k8sClient: config.K8sClient,
		logger: config.Logger.With(
			"resource", Name,
		),

		etcdCAFile:  config.EtcdCAFile,
		etcdCrtFile: config.EtcdCrtFile,
		etcdKeyFile: config.EtcdKeyFile,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) Underlying() framework.Resource {
	return r
}
