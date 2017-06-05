package flannel

import (
	"github.com/giantswarm/flanneltpr"
)

const (
	namespace = "giantswarm"
)

func app(spec flanneltpr.Spec) string {
	return "flannel-destroyer-" + clusterID(spec)
}

func clusterCustomer(spec flanneltpr.Spec) string {
	// TODO pass clusterCustomer with TPO
	return ""
}

func clusterID(spec flanneltpr.Spec) string {
	return clusterNamespace(spec)
}

func clusterName(spec flanneltpr.Spec) string {
	return clusterNamespace(spec)
}

func clusterNamespace(spec flanneltpr.Spec) string {
	return spec.Namespace
}

func hostPrivateNetwork(spec flanneltpr.Spec) string {
	return spec.Host.PrivateNetwork
}

func networkBridgeDockerImage(spec flanneltpr.Spec) string {
	// TODO add networkBridgeDockerImage to flanneltpr
	return "quay.io/giantswarm/k8s-network-bridge:201e9213e30af01c49f8ad30e5a7b9aed4755466"
}

func networkBridgeName(spec flanneltpr.Spec) string {
	return spec.Network.BridgeName
}

func networkEnvFilePath(spec flanneltpr.Spec) string {
	return spec.Network.EnvFilePath
}

func networkInterfaceName(spec flanneltpr.Spec) string {
	return spec.Network.InterfaceName
}
