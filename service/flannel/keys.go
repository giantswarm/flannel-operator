package flannel

import (
	"fmt"
	"strings"

	"github.com/giantswarm/flanneltpr"
)

const (
	creatorApp   = "flannel-creator"
	destroyerApp = "flannel-destroyer"
)

func creatorNamespace(spec flanneltpr.Spec) string {
	return creatorApp + "-" + clusterID(spec)
}

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
	return "coreos.com/network/" + networkBridgeName(spec)
}

func flannelDockerImage(spec flanneltpr.Spec) string {
	return spec.Flannel.Docker.Image
}

func hostPrivateNetwork(spec flanneltpr.Spec) string {
	return spec.Bridge.Config.PrivateNetwork
}

func networkBridgeDockerImage(spec flanneltpr.Spec) string {
	return spec.Bridge.Docker.Image
}

func networkBridgeName(spec flanneltpr.Spec) string {
	return spec.Bridge.Config.Name
}

func networkDNSBlock(spec flanneltpr.Spec) string {
	var parts []string
	for _, s := range spec.Bridge.Config.DNS.Servers {
		parts = append(parts, fmt.Sprintf("DNS=%s", s.String()))
	}
	return strings.Join(parts, "\n")
}

func networkEnvFilePath(spec flanneltpr.Spec) string {
	return fmt.Sprintf("/run/flannel/networks/%s.env", networkBridgeName(spec))
}

func networkInterfaceName(spec flanneltpr.Spec) string {
	return spec.Bridge.Config.Interface
}

func networkNTPBlock(spec flanneltpr.Spec) string {
	var parts []string
	for _, s := range spec.Bridge.Config.NTP.Servers {
		parts = append(parts, fmt.Sprintf("NTP=%s", s.String()))
	}
	return strings.Join(parts, "\n")
}
