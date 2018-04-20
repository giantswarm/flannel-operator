package controller

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/informer"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/flannel-operator/service/controller/v2"
	"github.com/giantswarm/flannel-operator/service/controller/v3"
)

type FrameworkConfig struct {
	CRDClient *k8scrdclient.CRDClient
	K8sClient kubernetes.Interface
	G8sClient versioned.Interface
	Logger    micrologger.Logger

	CAFile       string
	CrtFile      string
	EtcdEndpoint string
	KeyFile      string
	ProjectName  string
}

func NewFramework(config FrameworkConfig) (*controller.Controller, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.G8sClient must not be empty")
	}

	var err error

	var newInformer *informer.Informer
	{
		c := informer.Config{
			Watcher: config.G8sClient.CoreV1alpha1().FlannelConfigs(""),

			RateWait:     informer.DefaultRateWait,
			ResyncPeriod: informer.DefaultResyncPeriod,
		}

		newInformer, err = informer.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resourceRouter, err := newResourceRouter(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var crdFramework *controller.Controller
	{
		c := controller.Config{
			CRD:            v1alpha1.NewFlannelConfigCRD(),
			CRDClient:      config.CRDClient,
			Informer:       newInformer,
			Logger:         config.Logger,
			ResourceRouter: resourceRouter,
		}

		crdFramework, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return crdFramework, nil
}

func newResourceRouter(config FrameworkConfig) (*controller.ResourceRouter, error) {
	var err error

	var v2ResourceSet *controller.ResourceSet
	{
		c := v2.ResourceSetConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			CAFile:       config.CAFile,
			CrtFile:      config.CrtFile,
			EtcdEndpoint: config.EtcdEndpoint,
			KeyFile:      config.KeyFile,
			ProjectName:  config.ProjectName,
		}

		v2ResourceSet, err = v2.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v3ResourceSet *controller.ResourceSet
	{
		c := v3.ResourceSetConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			CAFile:       config.CAFile,
			CrtFile:      config.CrtFile,
			EtcdEndpoint: config.EtcdEndpoint,
			KeyFile:      config.KeyFile,
			ProjectName:  config.ProjectName,
		}

		v3ResourceSet, err = v3.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resourceRouter *controller.ResourceRouter
	{
		c := controller.ResourceRouterConfig{
			Logger: config.Logger,

			ResourceSets: []*controller.ResourceSet{
				v2ResourceSet,
				v3ResourceSet,
			},
		}

		resourceRouter, err = controller.NewResourceRouter(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resourceRouter, nil
}
