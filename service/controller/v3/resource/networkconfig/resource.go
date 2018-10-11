package networkconfig

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/flannel-operator/service/controller/v3/etcd"
)

const (
	// BackendType is the backend type the flannel network is configured with.
	BackendType = "vxlan"
	// Name is the identifier of the resource.
	Name = "networkconfigv3"
)

// Config represents the configuration used to create a new network config
// resource.
type Config struct {
	Logger micrologger.Logger
	Store  etcd.Store
}

// Resource implements the network config resource.
type Resource struct {
	logger micrologger.Logger
	store  etcd.Store
}

// New creates a new configured network config resource.
func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Store == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Store must not be empty", config)
	}

	r := &Resource{
		logger: config.Logger,
		store:  config.Store,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func toNetworkConfig(v interface{}) (NetworkConfig, error) {
	networkConfig, ok := v.(NetworkConfig)
	if !ok {
		return NetworkConfig{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &NetworkConfig{}, v)
	}

	return networkConfig, nil
}
