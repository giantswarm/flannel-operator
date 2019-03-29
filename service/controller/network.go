package controller

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/informer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/flannel-operator/service/controller/v3"
)

type NetworkConfig struct {
	CRDClient *k8scrdclient.CRDClient
	K8sClient kubernetes.Interface
	G8sClient versioned.Interface
	Logger    micrologger.Logger

	CAFile           string
	CrtFile          string
	CRDLabelSelector string
	EtcdEndpoints    []string
	KeyFile          string
	ProjectName      string
}

func (c NetworkConfig) newInformerListOptions() metav1.ListOptions {
	listOptions := metav1.ListOptions{
		LabelSelector: c.CRDLabelSelector,
	}

	return listOptions
}

type Network struct {
	*controller.Controller
}

func NewNetwork(config NetworkConfig) (*Network, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.G8sClient must not be empty")
	}

	var err error

	var newInformer *informer.Informer
	{
		c := informer.Config{
			Logger:  config.Logger,
			Watcher: config.G8sClient.CoreV1alpha1().FlannelConfigs(""),

			ListOptions:  config.newInformerListOptions(),
			RateWait:     informer.DefaultRateWait,
			ResyncPeriod: informer.DefaultResyncPeriod,
		}

		newInformer, err = informer.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resourceSets, err := newResourceSets(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			CRD:          v1alpha1.NewFlannelConfigCRD(),
			CRDClient:    config.CRDClient,
			Informer:     newInformer,
			Logger:       config.Logger,
			ResourceSets: resourceSets,
			RESTClient:   config.G8sClient.CoreV1alpha1().RESTClient(),

			Name: config.ProjectName,
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
			ProjectName:   config.ProjectName,
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
