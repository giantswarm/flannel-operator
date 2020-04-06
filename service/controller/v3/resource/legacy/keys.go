package legacy

import (
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
)

const (
	// networkApp is the app label for resources running flannel
	// components.
	networkApp = "flannel-network"
	// networkApp is the app label for resources cleaning flannel network
	// and bridges.
	destroyerApp = "flannel-destroyer"
)

// destroyerNamespace returns the namespace in which resources performing
// cleanup run in.
func destroyerNamespace(spec v1alpha1.FlannelConfigSpec) string {
	return destroyerApp + "-" + clusterID(spec)
}

func clusterID(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Cluster.ID
}

func clusterRoleBindingForDeletion(spec v1alpha1.FlannelConfigSpec) string {
	return clusterID(spec) + "-deletion"
}

func clusterRoleBinding(spec v1alpha1.FlannelConfigSpec) string {
	return networkApp + "-" + clusterID(spec)
}

func clusterRoleBindingForPodSecurityPolicy(spec v1alpha1.FlannelConfigSpec) string {
	return networkApp + "-" + clusterID(spec) + "-psp"
}

func clusterRoleBindingForPodSecurityPolicyForDeletion(spec v1alpha1.FlannelConfigSpec) string {
	return networkApp + "-" + clusterID(spec) + "-deletion-psp"
}

func flannelRunDir(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Flannel.Spec.RunDir
}

func hostPrivateNetwork(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Bridge.Spec.PrivateNetwork
}

func networkBridgeDockerImage(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Bridge.Docker.Image
}

func networkBridgeName(spec v1alpha1.FlannelConfigSpec) string {
	return "br-" + clusterID(spec)
}

func networkDNSBlock(spec v1alpha1.FlannelConfigSpec) string {
	var parts []string
	for _, s := range spec.Bridge.Spec.DNS.Servers {
		parts = append(parts, fmt.Sprintf("DNS=%s", s))
	}
	return strings.Join(parts, "\n")
}

func networkEnvFilePath(spec v1alpha1.FlannelConfigSpec) string {
	return fmt.Sprintf("%s/networks/%s.env", flannelRunDir(spec), networkBridgeName(spec))
}

func networkFlannelDevice(spec v1alpha1.FlannelConfigSpec) string {
	return fmt.Sprintf("flannel.%d", spec.Flannel.Spec.VNI)
}

func networkInterfaceName(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Bridge.Spec.Interface
}

func networkNTPBlock(spec v1alpha1.FlannelConfigSpec) string {
	var parts []string
	for _, s := range spec.Bridge.Spec.NTP.Servers {
		parts = append(parts, fmt.Sprintf("NTP=%s", s))
	}
	return strings.Join(parts, "\n")
}

func serviceAccountName(spec v1alpha1.FlannelConfigSpec) string {
	return clusterID(spec)
}
