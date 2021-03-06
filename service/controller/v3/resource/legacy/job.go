package legacy

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
)

func newJob(customObject v1alpha1.FlannelConfig, replicas int32) *batchv1.Job {
	privileged := true

	app := destroyerApp

	labels := map[string]string{
		"cluster":  key.ClusterID(customObject),
		"customer": key.ClusterCustomer(customObject),
		"app":      app,
	}

	return &batchv1.Job{
		TypeMeta: apismetav1.TypeMeta{
			Kind:       "deployment",
			APIVersion: "apps/v1",
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
					ServiceAccountName: serviceAccountName(customObject.Spec),
					RestartPolicy:      apiv1.RestartPolicyOnFailure,
					HostNetwork:        true,
					HostPID:            true,
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
									Path: "/etc/kubernetes/ssl/etcd/",
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
									Path: flannelRunDir(customObject.Spec),
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
							Image:           networkBridgeDockerImage(customObject.Spec),
							ImagePullPolicy: apiv1.PullAlways,
							Command: []string{
								"/bin/sh",
								"-c",
								"/docker-entrypoint.sh delete ${NETWORK_ENV_FILE_PATH} ${NETWORK_BRIDGE_NAME} ${NETWORK_INTERFACE_NAME} ${HOST_PRIVATE_NETWORK}",
							},
							Env: []apiv1.EnvVar{
								{
									Name:  "HOST_PRIVATE_NETWORK",
									Value: hostPrivateNetwork(customObject.Spec),
								},
								{
									Name:  "NETWORK_BRIDGE_NAME",
									Value: networkBridgeName(customObject.Spec),
								},
								{
									Name:  "NETWORK_DNS_BLOCK",
									Value: networkDNSBlock(customObject.Spec),
								},
								{
									Name:  "NETWORK_ENV_FILE_PATH",
									Value: networkEnvFilePath(customObject.Spec),
								},
								{
									Name:  "NETWORK_FLANNEL_DEVICE",
									Value: networkFlannelDevice(customObject.Spec),
								},
								{
									Name:  "NETWORK_INTERFACE_NAME",
									Value: networkInterfaceName(customObject.Spec),
								},
								{
									Name:  "NETWORK_NTP_BLOCK",
									Value: networkNTPBlock(customObject.Spec),
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
