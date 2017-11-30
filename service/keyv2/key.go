package keyv2

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
)

const (
	// NetworkID is the ID used to label apps for resources running flannel
	// components.
	NetworkID = "flannel-network"
)

func ClusterCustomer(customObject v1alpha1.Flannel) string {
	return customObject.Spec.Cluster.Customer
}

func ClusterID(customObject v1alpha1.Flannel) string {
	return customObject.Spec.Cluster.ID
}

func ClusterNamespace(customObject v1alpha1.Flannel) string {
	return customObject.Spec.Cluster.Namespace
}

func EtcdNetworkConfigPath(customObject v1alpha1.Flannel) string {
	return EtcdNetworkPath(customObject) + "/config"
}

func EtcdNetworkPath(customObject v1alpha1.Flannel) string {
	return "coreos.com/network/" + NetworkBridgeName(customObject)
}

func NetworkBridgeName(customObject v1alpha1.Flannel) string {
	return "br-" + ClusterID(customObject)
}

func NetworkNamespace(customObject v1alpha1.Flannel) string {
	return NetworkID + "-" + ClusterID(customObject)
}

func ToCustomObject(v interface{}) (v1alpha1.Flannel, error) {
	customObjectPointer, ok := v.(*v1alpha1.Flannel)
	if !ok {
		return v1alpha1.Flannel{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1alpha1.Flannel{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func VersionBundleVersion(customObject v1alpha1.Flannel) string {
	return customObject.Spec.VersionBundle.Version
}
