package flannel

import (
	apiunversioned "k8s.io/client-go/pkg/api/unversioned"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensionsv1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/giantswarm/flanneltpr"
)

func newDeployment(spec flanneltpr.Spec, replicas int32) *extensionsv1.Deployment {
	privileged := true

	return &extensionsv1.Deployment{
		TypeMeta: apiunversioned.TypeMeta{
			Kind:       "deployment",
			APIVersion: "extensions/v1beta",
		},
		ObjectMeta: apiv1.ObjectMeta{
			Name: app,
			Labels: map[string]string{
				"cluster":  clusterName(spec),
				"customer": clusterCustomer(spec),
				"app":      app,
			},
		},
		Spec: extensionsv1.DeploymentSpec{
			Strategy: extensionsv1.DeploymentStrategy{
				Type: extensionsv1.RecreateDeploymentStrategyType,
			},
			Replicas: &replicas,
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: apiv1.ObjectMeta{
					GenerateName: app,
					Labels: map[string]string{
						"cluster":  clusterName(spec),
						"customer": clusterCustomer(spec),
						"app":      app,
					},
					Annotations: map[string]string{
						"seccomp.security.alpha.kubernetes.io/pod": "unconfined",
					},
				},
				Spec: apiv1.PodSpec{
					HostNetwork: true,
					Volumes: []apiv1.Volume{
						{
							Name: "cgroup",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/sys/fs/cgroup",
								},
							},
						},
						{
							Name: "dbus",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/var/run/dbus",
								},
							},
						},
						{
							Name: "environment",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/etc/environment",
								},
							},
						},
						{
							Name: "etcd-certs",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/etc/giantswarm/g8s/ssl/etcd/",
								},
							},
						},
						{
							Name: "etc-systemd",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/etc/systemd/",
								},
							},
						},
						{
							Name: "flannel",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/run/flannel",
								},
							},
						},
						{
							Name: "ssl",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/etc/ssl/certs",
								},
							},
						},
						{
							Name: "systemctl",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/usr/bin/systemctl",
								},
							},
						},
						{
							Name: "systemd",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/run/systemd",
								},
							},
						},
						{
							Name: "sys-class-net",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: "/sys/class/net/",
								},
							},
						},
					},
					Containers: []apiv1.Container{
						{
							Name:            "k8s-network-bridge",
							Image:           networkBridgeDockerImage(spec),
							ImagePullPolicy: apiv1.PullAlways,
							Command: []string{
								"/bin/sh",
								"-c",
								"/docker-entrypoint.sh delete ${NETWORK_ENV_FILE_PATH} ${NETWORK_BRIDGE_NAME} ${NETWORK_INTERFACE_NAME} ${HOST_PRIVATE_NETWORK}",
							},
							Env: []apiv1.EnvVar{
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
									Name:  "NETWORK_INTERFACE_NAME",
									Value: networkInterfaceName(spec),
								},
							},
							SecurityContext: &apiv1.SecurityContext{
								Privileged: &privileged,
							},
							VolumeMounts: []apiv1.VolumeMount{
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
					},
				},
			},
		},
	}
}
