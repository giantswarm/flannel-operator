package flanneld

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
)

const (
	// Name is the identifier of the resource.
	Name = "flanneldv3"
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
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.EtcdCAFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.EtcdCAFile must not be empty", config)
	}
	if config.EtcdCrtFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.EtcdCrtFile must not be empty", config)
	}
	if config.EtcdKeyFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.EtcdKeyFile must not be empty", config)
	}

	r := &Resource{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		etcdCAFile:  config.EtcdCAFile,
		etcdCrtFile: config.EtcdCrtFile,
		etcdKeyFile: config.EtcdKeyFile,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func toDaemonSet(v interface{}) (*v1beta1.DaemonSet, error) {
	if v == nil {
		return nil, nil
	}

	daemonSet, ok := v.(*v1beta1.DaemonSet)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1beta1.DaemonSet{}, v)
	}

	return daemonSet, nil
}
