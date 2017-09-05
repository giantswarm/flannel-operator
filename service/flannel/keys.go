package flannel

import (
	"fmt"
	"strings"

	"github.com/giantswarm/flanneltpr"
)

const (
	// networkApp is the app label for resources running flannel
	// components.
	networkApp = "flannel-network"
	// networkApp is the app label for resources cleaning flannel network
	// and bridges.
	destroyerApp = "flannel-destroyer"
)

// networkNamespace returns the namespace in which the operator's resources run
// in.
func networkNamespace(spec flanneltpr.Spec) string {
	return networkApp + "-" + clusterID(spec)
}

// destroyerNamespace returns the namespace in which resources performing
// cleanup run in.
func destroyerNamespace(spec flanneltpr.Spec) string {
	return destroyerApp + "-" + clusterID(spec)
}

func clusterCustomer(spec flanneltpr.Spec) string {
	return spec.Cluster.Customer
}

func clusterID(spec flanneltpr.Spec) string {
	return spec.Cluster.ID
}

func clusterName(spec flanneltpr.Spec) string {
	return clusterID(spec)
}

func clusterNamespace(spec flanneltpr.Spec) string {
	return spec.Cluster.Namespace
}

func etcdPath(spec flanneltpr.Spec) string {
	return "coreos.com/network/" + networkBridgeName(spec) + "/config"
}

func flannelDockerImage(spec flanneltpr.Spec) string {
	return spec.Flannel.Docker.Image
}

func flannelRunDir(spec flanneltpr.Spec) string {
	return spec.Flannel.Spec.RunDir
}

func hostPrivateNetwork(spec flanneltpr.Spec) string {
	return spec.Bridge.Spec.PrivateNetwork
}

func networkBridgeDockerImage(spec flanneltpr.Spec) string {
	return spec.Bridge.Docker.Image
}

func networkBridgeName(spec flanneltpr.Spec) string {
	return "br-" + clusterID(spec)
}

func networkDNSBlock(spec flanneltpr.Spec) string {
	var parts []string
	for _, s := range spec.Bridge.Spec.DNS.Servers {
		parts = append(parts, fmt.Sprintf("DNS=%s", s.String()))
	}
	return strings.Join(parts, "\n")
}

func networkEnvFilePath(spec flanneltpr.Spec) string {
	return fmt.Sprintf("%s/networks/%s.env", flannelRunDir(spec), networkBridgeName(spec))
}

func networkFlannelDevice(spec flanneltpr.Spec) string {
	return fmt.Sprintf("flannel.%d", spec.Flannel.Spec.VNI)
}

func networkInterfaceName(spec flanneltpr.Spec) string {
	return spec.Bridge.Spec.Interface
}

func networkNTPBlock(spec flanneltpr.Spec) string {
	var parts []string
	for _, s := range spec.Bridge.Spec.NTP.Servers {
		parts = append(parts, fmt.Sprintf("NTP=%s", s))
	}
	return strings.Join(parts, "\n")
}
