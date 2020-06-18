package controller

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/flannel-operator/pkg/project"
	v3 "github.com/giantswarm/flannel-operator/service/controller/v3"
)

type NetworkConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	CAFile           string
	CrtFile          string
	CRDLabelSelector string
	EtcdEndpoints    []string
	KeyFile          string
}

type Network struct {
	*controller.Controller
}

func NewNetwork(config NetworkConfig) (*Network, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sClient must not be empty")
	}

	var err error

	resourceSets, err := newResourceSets(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			K8sClient:    config.K8sClient,
			Logger:       config.Logger,
			ResourceSets: resourceSets,
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha1.FlannelConfig)
			},

			Name: project.Name(),
		}

		operatorkitController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	n := &Network{
		Controller: operatorkitController,
	}

	return n, nil
}

func newResourceSets(config NetworkConfig) ([]*controller.ResourceSet, error) {
	var err error

	var v3ResourceSet *controller.ResourceSet
	{
		c := v3.ResourceSetConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			CAFile:        config.CAFile,
			CrtFile:       config.CrtFile,
			EtcdEndpoints: config.EtcdEndpoints,
			KeyFile:       config.KeyFile,
		}

		v3ResourceSet, err = v3.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resourceSets := []*controller.ResourceSet{
		v3ResourceSet,
	}

	return resourceSets, nil
}
