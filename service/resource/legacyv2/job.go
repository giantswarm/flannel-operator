package legacyv2

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newJob(spec v1alpha1.FlannelConfigSpec, replicas int32) *batchv1.Job {
	privileged := true

	app := destroyerApp

	labels := map[string]string{
		"cluster-id":  clusterName(spec),
		"customer-id": clusterCustomer(spec),
		"app":         app,
	}

	return &batchv1.Job{
		TypeMeta: apismetav1.TypeMeta{
			Kind:       "deployment",
			APIVersion: "extensions/v1beta",
		},
		ObjectMeta: apismetav1.ObjectMeta{
			Name:   app,
			Labels: labels,
		},
		Spec: batchv1.JobSpec{
			Parallelism: &replicas,
			Completions: &replicas,
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: apismetav1.ObjectMeta{
					GenerateName: app,
					Labels:       labels,
				},
				Spec: apiv1.PodSpec{
					RestartPolicy:      apiv1.RestartPolicyOnFailure,
					HostNetwork:        true,
					ServiceAccountName: "flannel-operator",
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
									Path: flannelRunDir(spec),
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
