package flannel

import (
	"github.com/giantswarm/flanneltpr"
	apismeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	apisextensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"

	api "k8s.io/client-go/pkg/api/v1"
)

func newDaemonSet(spec flanneltpr.Spec) *apisextensions.DaemonSet {
	app := networkApp

	labels := map[string]string{
		"cluster":  clusterName(spec),
		"customer": clusterCustomer(spec),
		"app":      app,
	}

	containers := newDaemonSetContainers(spec)
	volumes := newDaemonSetVolumes(spec)

	return &apisextensions.DaemonSet{
		TypeMeta: apismeta.TypeMeta{
			Kind:       "daemonset",
			APIVersion: "extensions/v1beta",
		},
		ObjectMeta: apismeta.ObjectMeta{
			Name:   app,
			Labels: labels,
		},
		Spec: apisextensions.DaemonSetSpec{
			Template: api.PodTemplateSpec{
				ObjectMeta: apismeta.ObjectMeta{
					GenerateName: creatorApp,
					Labels:       labels,
					Annotations: map[string]string{
						"seccomp.security.alpha.kubernetes.io/pod": "unconfined",
					},
				},
				Spec: api.PodSpec{
					HostNetwork: true,
					Containers:  containers,
					Volumes:     volumes,
				},
			},
		},
	}
}

func newDaemonSetContainers(spec flanneltpr.Spec) []api.Container {
	privileged := true

	return []api.Container{
		{
			Name:            "flanneld",
			Image:           flannelDockerImage(spec),
			ImagePullPolicy: api.PullAlways,
			Command: []string{
				"/bin/sh",
				"-c",
				"/opt/bin/flanneld --etcd-endpoints https://127.0.0.1:2379 --public-ip=$NODE_IP --iface=$NODE_IP --networks=$NETWORK_BRIDGE_NAME -v=0",
			},
			Env: []api.EnvVar{
				{
					Name:  "NETWORK_BRIDGE_NAME",
					Value: networkBridgeName(spec),
				},
				{
					Name: "NODE_IP",
					ValueFrom: &api.EnvVarSource{
						FieldRef: &api.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "spec.nodeName",
						},
					},
				},
			},
			VolumeMounts: []api.VolumeMount{
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
					Name:  "NETWORK_ENV_FILE_PATH",
					Value: networkEnvFilePath(spec),
				},
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
					Name:  "NETWORK_NTP_BLOCK",
					Value: networkNTPBlock(spec),
				},
				{
					Name:  "NETWORK_INTERFACE_NAME",
					Value: networkInterfaceName(spec),
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
					Name:      "systemctl",
					MountPath: "/usr/bin/systemctl",
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
	}
}

func newDaemonSetVolumes(spec flanneltpr.Spec) []api.Volume {
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
					Path: "/etc/giantswarm/g8s/ssl/etcd/",
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
			Name: "systemctl",
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: "/usr/bin/systemctl",
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
