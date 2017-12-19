package keyv2

import (
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
)

const (
	// NetworkID is the ID used to label apps for resources running flannel
	// components.
	NetworkID = "flannel-network"
)

func ClusterCustomer(customObject v1alpha1.FlannelConfig) string {
	return customObject.Spec.Cluster.Customer
}

func ClusterID(customObject v1alpha1.FlannelConfig) string {
	return customObject.Spec.Cluster.ID
}

func ClusterNamespace(customObject v1alpha1.FlannelConfig) string {
	return customObject.Spec.Cluster.Namespace
}

func EtcdNetworkConfigPath(customObject v1alpha1.FlannelConfig) string {
	return EtcdNetworkPath(customObject) + "/config"
}

func EtcdNetworkPath(customObject v1alpha1.FlannelConfig) string {
	return "coreos.com/network/" + NetworkBridgeName(customObject)
}

func EtcdPrefix(customObject v1alpha1.FlannelConfig) string {
	return "/" + EtcdNetworkPath(customObject)
}

func FlannelRunDir(customObject v1alpha1.FlannelConfig) string {
	return customObject.Spec.Flannel.Spec.RunDir
}

func FlannelVNI(customObject v1alpha1.FlannelConfig) int {
	return customObject.Spec.Flannel.Spec.VNI
}

func HostPrivateNetwork(customObject v1alpha1.FlannelConfig) string {
	return customObject.Spec.Bridge.Spec.PrivateNetwork
}

func NetworkBridgeDockerImage(customObject v1alpha1.FlannelConfig) string {
	return customObject.Spec.Bridge.Docker.Image
}

func NetworkBridgeName(customObject v1alpha1.FlannelConfig) string {
	return "br-" + ClusterID(customObject)
}

func NetworkDNSBlock(customObject v1alpha1.FlannelConfig) string {
	var parts []string

	for _, s := range customObject.Spec.Bridge.Spec.DNS.Servers {
		parts = append(parts, fmt.Sprintf("DNS=%s", s))
	}

	return strings.Join(parts, "\n")
}

func NetworkEnvFilePath(customObject v1alpha1.FlannelConfig) string {
	return fmt.Sprintf("%s/networks/%s.env", FlannelRunDir(customObject), NetworkBridgeName(customObject))
}

func NetworkFlannelDevice(customObject v1alpha1.FlannelConfig) string {
	return fmt.Sprintf("flannel.%d", FlannelVNI(customObject))
}

func NetworkHealthDockerImage(customObject v1alpha1.FlannelConfig) string {
	return customObject.Spec.Health.Docker.Image
}

func NetworkInterfaceName(customObject v1alpha1.FlannelConfig) string {
	return customObject.Spec.Bridge.Spec.Interface
}

func NetworkNamespace(customObject v1alpha1.FlannelConfig) string {
	return NetworkID + "-" + ClusterID(customObject)
}

func NetworkNTPBlock(customObject v1alpha1.FlannelConfig) string {
	var parts []string

	for _, s := range customObject.Spec.Bridge.Spec.NTP.Servers {
		parts = append(parts, fmt.Sprintf("NTP=%s", s))
	}

	return strings.Join(parts, "\n")
}

func NetworkTapName(customObject v1alpha1.FlannelConfig) string {
	return "tap-" + ClusterID(customObject)
}

func ToCustomObject(v interface{}) (v1alpha1.FlannelConfig, error) {
	customObjectPointer, ok := v.(*v1alpha1.FlannelConfig)
	if !ok {
		return v1alpha1.FlannelConfig{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &v1alpha1.FlannelConfig{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func VersionBundleVersion(customObject v1alpha1.FlannelConfig) string {
	return customObject.Spec.VersionBundle.Version
}
