package legacyv2

import (
	"fmt"
	"strconv"
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
	// base port for liveness probes
	portBase = 21000
	// health endpoint
	healthEndpoint = "/healthz"
	// liveness probe host
	probeHost = "127.0.0.1"
	// liveness config
	initialDelaySeconds = 10
	timeoutSeconds      = 5
	periodSeconds       = 10
	failureThreshold    = 2
	successThreshold    = 1
)

// networkNamespace returns the namespace in which the operator's resources run
// in.
func networkNamespace(spec v1alpha1.FlannelConfigSpec) string {
	return networkApp + "-" + clusterID(spec)
}

// destroyerNamespace returns the namespace in which resources performing
// cleanup run in.
func destroyerNamespace(spec v1alpha1.FlannelConfigSpec) string {
	return destroyerApp + "-" + clusterID(spec)
}

func clusterCustomer(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Cluster.Customer
}

func clusterID(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Cluster.ID
}

func clusterName(spec v1alpha1.FlannelConfigSpec) string {
	return clusterID(spec)
}

func clusterNamespace(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Cluster.Namespace
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

func etcdNetworkConfigPath(spec v1alpha1.FlannelConfigSpec) string {
	return etcdNetworkPath(spec) + "/config"
}

func etcdNetworkPath(spec v1alpha1.FlannelConfigSpec) string {
	return "coreos.com/network/" + networkBridgeName(spec)
}

func etcdPrefix(spec v1alpha1.FlannelConfigSpec) string {
	return "/" + etcdNetworkPath(spec)
}

func flannelRunDir(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Flannel.Spec.RunDir
}

func healthListenAddress(spec v1alpha1.FlannelConfigSpec) string {
	return "http://" + probeHost + ":" + strconv.Itoa(int(livenessPort(spec)))
}
func hostPrivateNetwork(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Bridge.Spec.PrivateNetwork
}

func livenessPort(spec v1alpha1.FlannelConfigSpec) int32 {
	return int32(portBase + spec.Flannel.Spec.VNI)
}

func networkBridgeDockerImage(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Bridge.Docker.Image
}
func networkHealthDockerImage(spec v1alpha1.FlannelConfigSpec) string {
	return spec.Health.Docker.Image
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

func networkTapName(spec v1alpha1.FlannelConfigSpec) string {
	return "tap-" + clusterID(spec)
}

func serviceAccountName(spec v1alpha1.FlannelConfigSpec) string {
	return clusterID(spec)
}
