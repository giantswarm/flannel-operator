package keyv1

import (
	"github.com/giantswarm/flanneltpr"
	"github.com/giantswarm/microerror"
)

const (
	// NetworkID is the ID used to label apps for resources running flannel
	// components.
	NetworkID = "flannel-network"
)

func ClusterCustomer(customObject flanneltpr.CustomObject) string {
	return customObject.Spec.Cluster.Customer
}

func ClusterID(customObject flanneltpr.CustomObject) string {
	return customObject.Spec.Cluster.ID
}

func ClusterNamespace(customObject flanneltpr.CustomObject) string {
	return customObject.Spec.Cluster.Namespace
}

func EtcdNetworkConfigPath(customObject flanneltpr.CustomObject) string {
	return EtcdNetworkPath(customObject) + "/config"
}

func EtcdNetworkPath(customObject flanneltpr.CustomObject) string {
	return "coreos.com/network/" + NetworkBridgeName(customObject)
}

func NetworkBridgeName(customObject flanneltpr.CustomObject) string {
	return "br-" + ClusterID(customObject)
}

func NetworkNamespace(customObject flanneltpr.CustomObject) string {
	return NetworkID + "-" + ClusterID(customObject)
}

func ToCustomObject(v interface{}) (flanneltpr.CustomObject, error) {
	customObjectPointer, ok := v.(*flanneltpr.CustomObject)
	if !ok {
		return flanneltpr.CustomObject{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &flanneltpr.CustomObject{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func VersionBundleVersion(customObject flanneltpr.CustomObject) string {
	return customObject.Spec.VersionBundle.Version
}
