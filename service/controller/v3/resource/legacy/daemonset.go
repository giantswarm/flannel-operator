package legacy

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	api "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	apismeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
)

const (
	MAX_UNAVAILABLE = 1
)

func newDaemonSet(customObject v1alpha1.FlannelConfig, etcdCAFile, etcdCrtFile, etcdKeyFile string) *v1beta1.DaemonSet {
	app := networkApp

	containers := newDaemonSetContainers(customObject.Spec, etcdCAFile, etcdCrtFile, etcdKeyFile)
	volumes := newDaemonSetVolumes(customObject.Spec)
	maxUnavailable := intstr.FromInt(MAX_UNAVAILABLE)

	daemonSet := &v1beta1.DaemonSet{
		TypeMeta: apismeta.TypeMeta{
			Kind:       "daemonset",
			APIVersion: "extensions/v1beta",
		},
		ObjectMeta: apismeta.ObjectMeta{
			Name: app,
			Annotations: map[string]string{
				VersionBundleVersionAnnotation: key.VersionBundleVersion(customObject),
			},
			Labels: map[string]string{
				"app":      app,
				"cluster":  key.ClusterID(customObject),
				"customer": key.ClusterCustomer(customObject),
			},
		},
		Spec: v1beta1.DaemonSetSpec{
			Template: api.PodTemplateSpec{
				ObjectMeta: apismeta.ObjectMeta{
					GenerateName: app,
					Labels: map[string]string{
						"app":      app,
						"cluster":  key.ClusterID(customObject),
						"customer": key.ClusterCustomer(customObject),
					},
				},
				Spec: api.PodSpec{
					ServiceAccountName: serviceAccountName(customObject.Spec),
					HostNetwork:        true,
					Containers:         containers,
					Volumes:            volumes,
				},
			},
			UpdateStrategy: v1beta1.DaemonSetUpdateStrategy{
				Type: v1beta1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &v1beta1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
			},
		},
	}

	return daemonSet
}

func newDaemonSetContainers(spec v1alpha1.FlannelConfigSpec, etcdCAFile, etcdCrtFile, etcdKeyFile string) []api.Container {
	privileged := true

	return []api.Container{
		{
			Name:            "flanneld",
			Image:           key.FlannelDockerImage,
			ImagePullPolicy: api.PullAlways,
			Command: []string{
				"/bin/sh",
				"-c",
				"/opt/bin/flanneld --etcd-endpoints=https://127.0.0.1:2379 --etcd-cafile=${ETCD_CA} --etcd-certfile=${ETCD_CRT} --etcd-keyfile=${ETCD_KEY} --etcd-prefix=${ETCD_PREFIX} --iface=${NETWORK_INTERFACE_NAME} --subnet-file=${NETWORK_ENV_FILE_PATH} -v=0",
			},
			Env: []api.EnvVar{
				{
					Name:  "ETCD_CA",
					Value: etcdCAFile,
				},
				{
					Name:  "ETCD_CRT",
					Value: etcdCrtFile,
				},
				{
					Name:  "ETCD_KEY",
					Value: etcdKeyFile,
				},
				{
					Name:  "ETCD_PREFIX",
					Value: etcdPrefix(spec),
				},
				{
					Name:  "NETWORK_BRIDGE_NAME",
					Value: networkBridgeName(spec),
				},
				{
					Name:  "NETWORK_ENV_FILE_PATH",
					Value: networkEnvFilePath(spec),
				},
				{
					Name:  "NETWORK_INTERFACE_NAME",
					Value: networkInterfaceName(spec),
				},
			},
			LivenessProbe: &api.Probe{
				InitialDelaySeconds: initialDelaySeconds,
				TimeoutSeconds:      timeoutSeconds,
				PeriodSeconds:       periodSeconds,
				FailureThreshold:    failureThreshold,
				SuccessThreshold:    successThreshold,
				Handler: api.Handler{
					HTTPGet: &api.HTTPGetAction{
						Path: healthEndpoint,
						Port: intstr.IntOrString{IntVal: livenessPort(spec)},
						Host: probeHost,
					},
				},
			},
			VolumeMounts: []api.VolumeMount{
				{
					Name:      "etcd-certs",
					MountPath: "/etc/kubernetes/ssl/etcd",
				},
				{
					Name:      "flannel",
					MountPath: "/run/flannel",
				},
				{
					Name:      "ssl",
					MountPath: "/etc/ssl/certs",
				},
			},
			SecurityContext: &api.SecurityContext{
				Privileged: &privileged,
			},
		},
		{
			Name:            "k8s-network-bridge",
			Image:           networkBridgeDockerImage(spec),
			ImagePullPolicy: api.PullAlways,
			Command: []string{
				"/bin/sh",
				"-c",
				"while [ ! -f ${NETWORK_ENV_FILE_PATH} ]; do echo \"Waiting for ${NETWORK_ENV_FILE_PATH} to be created\"; sleep 1; done; /docker-entrypoint.sh create ${NETWORK_ENV_FILE_PATH} ${NETWORK_BRIDGE_NAME} ${NETWORK_INTERFACE_NAME} ${HOST_PRIVATE_NETWORK}",
			},
			Env: []api.EnvVar{
				{
					Name:  "HOST_PRIVATE_NETWORK",
					Value: hostPrivateNetwork(spec),
				},
				{
					Name:  "NETWORK_BRIDGE_NAME",
					Value: networkBridgeName(spec),
				},
				{
					Name:  "NETWORK_DNS_BLOCK",
					Value: networkDNSBlock(spec),
				},
				{
					Name:  "NETWORK_ENV_FILE_PATH",
					Value: networkEnvFilePath(spec),
				},
				{
					Name:  "NETWORK_FLANNEL_DEVICE",
					Value: networkFlannelDevice(spec),
				},
				{
					Name:  "NETWORK_INTERFACE_NAME",
					Value: networkInterfaceName(spec),
				},
				{
					Name:  "NETWORK_NTP_BLOCK",
					Value: networkNTPBlock(spec),
				},
				{
					Name:  "NETWORK_TAP_NAME",
					Value: networkTapName(spec),
				},
			},
			LivenessProbe: &api.Probe{
				InitialDelaySeconds: initialDelaySeconds,
				TimeoutSeconds:      timeoutSeconds,
				PeriodSeconds:       periodSeconds,
				FailureThreshold:    failureThreshold,
				SuccessThreshold:    successThreshold,
				Handler: api.Handler{
					HTTPGet: &api.HTTPGetAction{
						Path: healthEndpoint,
						Port: intstr.IntOrString{IntVal: livenessPort(spec)},
						Host: probeHost,
					},
				},
			},
			SecurityContext: &api.SecurityContext{
				Privileged: &privileged,
			},
			VolumeMounts: []api.VolumeMount{
				{
					Name:      "cgroup",
					MountPath: "/sys/fs/cgroup",
				},
				{
					Name:      "dbus",
					MountPath: "/var/run/dbus",
				},
				{
					Name:      "environment",
					MountPath: "/etc/environment",
				},
				{
					Name:      "etc-systemd",
					MountPath: "/etc/systemd/",
				},
				{
					Name:      "flannel",
					MountPath: "/run/flannel",
				},
				{
					Name:      "systemd",
					MountPath: "/run/systemd",
				},
				{
					Name:      "sys-class-net",
					MountPath: "/sys/class/net/",
				},
			},
		},
		{
			Name:            "flannel-network-health",
			Image:           networkHealthDockerImage(spec),
			ImagePullPolicy: api.PullAlways,
			Env: []api.EnvVar{
				{
					Name:  "LISTEN_ADDRESS",
					Value: healthListenAddress(spec),
				},
				{
					Name:  "NETWORK_BRIDGE_NAME",
					Value: networkBridgeName(spec),
				},
				{
					Name:  "NETWORK_ENV_FILE_PATH",
					Value: networkEnvFilePath(spec),
				},
				{
					Name:  "NETWORK_FLANNEL_DEVICE",
					Value: networkFlannelDevice(spec),
				},
			},
			SecurityContext: &api.SecurityContext{
				Privileged: &privileged,
			},
			VolumeMounts: []api.VolumeMount{
				{
					Name:      "flannel",
					MountPath: "/run/flannel",
				},
			},
		},
	}
}

func newDaemonSetVolumes(spec v1alpha1.FlannelConfigSpec) []api.Volume {
	return []api.Volume{
		{
			Name: "cgroup",
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: "/sys/fs/cgroup",
				},
			},
		},
		{
			Name: "dbus",
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: "/var/run/dbus",
				},
			},
		},
		{
			Name: "environment",
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: "/etc/environment",
				},
			},
		},
		{
			Name: "etcd-certs",
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: "/etc/kubernetes/ssl/etcd",
				},
			},
		},
		{
			Name: "etc-systemd",
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: "/etc/systemd/",
				},
			},
		},
		{
			Name: "flannel",
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: flannelRunDir(spec),
				},
			},
		},
		{
			Name: "ssl",
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: "/etc/ssl/certs",
				},
			},
		},
		{
			Name: "systemd",
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: "/run/systemd",
				},
			},
		},
		{
			Name: "sys-class-net",
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: "/sys/class/net/",
				},
			},
		},
	}
}
