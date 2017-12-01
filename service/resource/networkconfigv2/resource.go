package networkconfigv2

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/giantswarm/flannel-operator/service/etcdv2"
)

const (
	// BackendType is the backend type the flannel network is configured with.
	BackendType = "vxlan"
	// Name is the identifier of the resource.
	Name = "networkconfigv2"
)

// Config represents the configuration used to create a new network config
// resource.
type Config struct {
	Logger micrologger.Logger
	Store  etcdv2.Store
}

// DefaultConfig provides a default configuration to create a new network config
// resource by best effort.
func DefaultConfig() Config {
	return Config{
		Logger: nil,
		Store:  nil,
	}
}

// Resource implements the network config resource.
type Resource struct {
	logger micrologger.Logger
	store  etcdv2.Store
}

// New creates a new configured network config resource.
func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}
	if config.Store == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Store must not be empty")
	}

	newResource := &Resource{
		logger: config.Logger.With(
			"resource", Name,
		),
		store: config.Store,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) Underlying() framework.Resource {
	return r
}

func toNetworkConfig(v interface{}) (NetworkConfig, error) {
	networkConfig, ok := v.(NetworkConfig)
	if !ok {
		return NetworkConfig{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &NetworkConfig{}, v)
	}

	return networkConfig, nil
}
