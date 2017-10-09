package key

import (
	"github.com/giantswarm/flanneltpr"
	"github.com/giantswarm/microerror"
)

func ClusterID(customObject flanneltpr.CustomObject) string {
	return customObject.Spec.Cluster.ID
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

func ToCustomObject(v interface{}) (flanneltpr.CustomObject, error) {
	customObjectPointer, ok := v.(*flanneltpr.CustomObject)
	if !ok {
		return flanneltpr.CustomObject{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &flanneltpr.CustomObject{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}
