package v3

import (
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/controller/resource/metricsresource"
	"github.com/giantswarm/operatorkit/controller/resource/retryresource"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
	"github.com/giantswarm/flannel-operator/service/controller/v3/resource/flanneld"
	"github.com/giantswarm/flannel-operator/service/controller/v3/resource/legacy"
	"github.com/giantswarm/flannel-operator/service/controller/v3/resource/namespace"
)

type ResourceSetConfig struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	CAFile       string
	CrtFile      string
	EtcdEndpoint string
	KeyFile      string
	ProjectName  string
}

func NewResourceSet(config ResourceSetConfig) (*controller.ResourceSet, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sClient must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	if config.CrtFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.CrtFile must not be empty")
	}
	if config.EtcdEndpoint == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.EtcdEndpoint must not be empty")
	}
	if config.KeyFile == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.KeyFile must not be empty")
	}
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.ProjectName must not be empty")
	}

	var err error

	var flanneldResource controller.Resource
	{
		c := flanneld.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			EtcdCAFile:  config.CAFile,
			EtcdCrtFile: config.CrtFile,
			EtcdKeyFile: config.KeyFile,
		}

		ops, err := flanneld.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		flanneldResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var legacyResource controller.Resource
	{
		legacyConfig := legacy.DefaultConfig()

		legacyConfig.BackOff = backoff.NewExponential(5*time.Minute, 1*time.Minute)
		legacyConfig.K8sClient = config.K8sClient
		legacyConfig.Logger = config.Logger

		legacyConfig.EtcdCAFile = config.CAFile
		legacyConfig.EtcdCrtFile = config.CrtFile
		legacyConfig.EtcdKeyFile = config.KeyFile

		ops, err := legacy.New(legacyConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		legacyResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceResource controller.Resource
	{
		c := namespace.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		ops, err := namespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		namespaceResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []controller.Resource{
		namespaceResource,
		legacyResource,
		flanneldResource,
	}

	{
		c := retryresource.WrapConfig{
			Logger: config.Logger,
		}

		resources, err = retryresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		c := metricsresource.WrapConfig{}

		resources, err = metricsresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	handlesFunc := func(obj interface{}) bool {
		customObject, err := key.ToCustomObject(obj)
		if err != nil {
			return false
		}

		if key.VersionBundleVersion(customObject) == VersionBundle().Version {
			return true
		}

		return false
	}

	var resourceSet *controller.ResourceSet
	{
		c := controller.ResourceSetConfig{
			Handles:   handlesFunc,
			Logger:    config.Logger,
			Resources: resources,
		}

		resourceSet, err = controller.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resourceSet, nil
}

func toCRUDResource(logger micrologger.Logger, ops controller.CRUDResourceOps) (*controller.CRUDResource, error) {
	c := controller.CRUDResourceConfig{
		Logger: logger,
		Ops:    ops,
	}

	r, err := controller.NewCRUDResource(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r, nil
}
