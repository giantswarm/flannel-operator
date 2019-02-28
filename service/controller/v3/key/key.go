package key

import (
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// NetworkID is the ID used to label apps for resources running flannel
	// components.
	NetworkID = "flannel-network"

	// flanneld image
	FlannelDockerImage = "quay.io/giantswarm/flannel:v0.11.0-amd64"
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

func KubePrefix(customObject v1alpha1.FlannelConfig) string {
	return ClusterID(customObject)
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

func IsDeleted(customObject v1alpha1.FlannelConfig) bool {
	return customObject.GetDeletionTimestamp() != nil
}

// MaxUnavailable is used for the Kubernetes update strategy. We want only one
// pod at a time to be unavailable during updates.
func MaxUnavailable() *intstr.IntOrString {
	v := intstr.FromInt(1)
	return &v
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

func ServiceAccountName(customResource v1alpha1.FlannelConfig) string {
	return ClusterID(customResource)
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
