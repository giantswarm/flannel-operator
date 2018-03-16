package flanneld

import (
	"context"
	"strconv"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	apismeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/giantswarm/flannel-operator/service/flannelconfig/v3/key"
)

var (
	containersPrivileged       = true
	failureThreshold     int32 = 2
	healthEndpoint             = "/healthz"
	initialDelaySeconds  int32 = 10
	periodSeconds        int32 = 10
	portBase                   = 21000
	probeHost                  = "127.0.0.1"
	successThreshold     int32 = 1
	timeoutSeconds       int32 = 5
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "debug", "computing the desired daemon set")

	daemonSet := newDaemonSet(customObject, r.etcdCAFile, r.etcdCrtFile, r.etcdKeyFile)

	r.logger.LogCtx(ctx, "debug", "computed the desired daemon set")

	return daemonSet, nil
}

func healthListenAddress(customObject v1alpha1.FlannelConfig) string {
	return "http://" + probeHost + ":" + strconv.Itoa(int(livenessProbePort(customObject)))
}

func livenessProbePort(customObject v1alpha1.FlannelConfig) int32 {
	return int32(portBase + key.FlannelVNI(customObject))
}

func newDaemonSet(customObject v1alpha1.FlannelConfig, etcdCAFile, etcdCrtFile, etcdKeyFile string) *v1beta1.DaemonSet {
	return &v1beta1.DaemonSet{
		TypeMeta: apismeta.TypeMeta{
			Kind:       "daemonset",
			APIVersion: "extensions/v1beta",
		},
		ObjectMeta: apismeta.ObjectMeta{
			Name: key.NetworkID,
			Annotations: map[string]string{
				VersionBundleVersionAnnotation: key.VersionBundleVersion(customObject),
			},
			Labels: map[string]string{
				"app":      key.NetworkID,
				"cluster":  key.ClusterID(customObject),
				"customer": key.ClusterCustomer(customObject),
			},
		},
		Spec: v1beta1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: apismeta.ObjectMeta{
					GenerateName: key.NetworkID,
					Labels: map[string]string{
						"app":      key.NetworkID,
						"cluster":  key.ClusterID(customObject),
						"customer": key.ClusterCustomer(customObject),
					},
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					Containers: []corev1.Container{
						{
							Name:            "flanneld",
							Image:           "quay.io/giantswarm/flannel:v0.9.0-amd64",
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"/bin/sh",
								"-c",
								"/opt/bin/flanneld --etcd-endpoints=https://127.0.0.1:2379 --etcd-cafile=${ETCD_CA} --etcd-certfile=${ETCD_CRT} --etcd-keyfile=${ETCD_KEY} --etcd-prefix=${ETCD_PREFIX} --iface=${NETWORK_INTERFACE_NAME} --subnet-file=${NETWORK_ENV_FILE_PATH} -v=0",
							},
							Env: []corev1.EnvVar{
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
									Value: key.EtcdPrefix(customObject),
								},
								{
									Name:  "NETWORK_BRIDGE_NAME",
									Value: key.NetworkBridgeName(customObject),
								},
								{
									Name:  "NETWORK_ENV_FILE_PATH",
									Value: key.NetworkEnvFilePath(customObject),
								},
								{
									Name:  "NETWORK_INTERFACE_NAME",
									Value: key.NetworkInterfaceName(customObject),
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: initialDelaySeconds,
								TimeoutSeconds:      timeoutSeconds,
								PeriodSeconds:       periodSeconds,
								FailureThreshold:    failureThreshold,
								SuccessThreshold:    successThreshold,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: healthEndpoint,
										Port: intstr.IntOrString{IntVal: livenessProbePort(customObject)},
										Host: probeHost,
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
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
							SecurityContext: &corev1.SecurityContext{
								Privileged: &containersPrivileged,
							},
						},
						{
							Name:            "k8s-network-bridge",
							Image:           key.NetworkBridgeDockerImage(customObject),
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"/bin/sh",
								"-c",
								"while [ ! -f ${NETWORK_ENV_FILE_PATH} ]; do echo \"Waiting for ${NETWORK_ENV_FILE_PATH} to be created\"; sleep 1; done; /docker-entrypoint.sh create ${NETWORK_ENV_FILE_PATH} ${NETWORK_BRIDGE_NAME} ${NETWORK_INTERFACE_NAME} ${HOST_PRIVATE_NETWORK}",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "HOST_PRIVATE_NETWORK",
									Value: key.HostPrivateNetwork(customObject),
								},
								{
									Name:  "NETWORK_BRIDGE_NAME",
									Value: key.NetworkBridgeName(customObject),
								},
								{
									Name:  "NETWORK_DNS_BLOCK",
									Value: key.NetworkDNSBlock(customObject),
								},
								{
									Name:  "NETWORK_ENV_FILE_PATH",
									Value: key.NetworkEnvFilePath(customObject),
								},
								{
									Name:  "NETWORK_FLANNEL_DEVICE",
									Value: key.NetworkFlannelDevice(customObject),
								},
								{
									Name:  "NETWORK_INTERFACE_NAME",
									Value: key.NetworkInterfaceName(customObject),
								},
								{
									Name:  "NETWORK_NTP_BLOCK",
									Value: key.NetworkNTPBlock(customObject),
								},
								{
									Name:  "NETWORK_TAP_NAME",
									Value: key.NetworkTapName(customObject),
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: initialDelaySeconds,
								TimeoutSeconds:      timeoutSeconds,
								PeriodSeconds:       periodSeconds,
								FailureThreshold:    failureThreshold,
								SuccessThreshold:    successThreshold,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: healthEndpoint,
										Port: intstr.IntOrString{IntVal: livenessProbePort(customObject)},
										Host: probeHost,
									},
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &containersPrivileged,
							},
							VolumeMounts: []corev1.VolumeMount{
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
							Image:           key.NetworkHealthDockerImage(customObject),
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{
									Name:  "LISTEN_ADDRESS",
									Value: healthListenAddress(customObject),
								},
								{
									Name:  "NETWORK_BRIDGE_NAME",
									Value: key.NetworkBridgeName(customObject),
								},
								{
									Name:  "NETWORK_ENV_FILE_PATH",
									Value: key.NetworkEnvFilePath(customObject),
								},
								{
									Name:  "NETWORK_FLANNEL_DEVICE",
									Value: key.NetworkFlannelDevice(customObject),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &containersPrivileged,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "flannel",
									MountPath: "/run/flannel",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "cgroup",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys/fs/cgroup",
								},
							},
						},
						{
							Name: "dbus",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run/dbus",
								},
							},
						},
						{
							Name: "environment",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/environment",
								},
							},
						},
						{
							Name: "etcd-certs",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/giantswarm/g8s/ssl/etcd",
								},
							},
						},
						{
							Name: "etc-systemd",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/systemd/",
								},
							},
						},
						{
							Name: "flannel",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: key.FlannelRunDir(customObject),
								},
							},
						},
						{
							Name: "ssl",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/ssl/certs",
								},
							},
						},
						{
							Name: "systemd",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/systemd",
								},
							},
						},
						{
							Name: "sys-class-net",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys/class/net/",
								},
							},
						},
					},
				},
			},
		},
	}
}
