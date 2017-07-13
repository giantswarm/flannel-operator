package flannel

import (
	"fmt"

	"github.com/giantswarm/flanneltpr"
)

const (
	destroyerApp = "flannel-destroyer"
)

func destroyerNamespace(spec flanneltpr.Spec) string {
	return destroyerApp + "-" + clusterID(spec)
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
	return spec.Network.Bridge.Docker.Image
}

func networkBridgeName(spec flanneltpr.Spec) string {
	return spec.Network.BridgeName
}

func networkEnvFilePath(spec flanneltpr.Spec) string {
	return spec.Network.EnvFilePath
}

// TODO use Flannel VNI from spec.
func networkFlannelDevice(spec flanneltpr.Spec) string {
	return fmt.Sprintf("flannel.%d", "TODO")
}

func networkInterfaceName(spec flanneltpr.Spec) string {
	return spec.Network.InterfaceName
}
