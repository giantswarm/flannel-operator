package flanneld

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
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

	r.logger.LogCtx(ctx, "level", "debug", "message", "computing the desired daemon set")

	daemonSet := newDaemonSet(customObject, r.etcdEndpoints, r.etcdCAFile, r.etcdCrtFile, r.etcdKeyFile)

	r.logger.LogCtx(ctx, "level", "debug", "message", "computed the desired daemon set")

	return daemonSet, nil
}

func healthListenAddress(customObject v1alpha1.FlannelConfig) string {
	return "http://" + probeHost + ":" + strconv.Itoa(int(livenessProbePort(customObject)))
}

func livenessProbePort(customObject v1alpha1.FlannelConfig) int32 {
	return int32(portBase + key.FlannelVNI(customObject))
}

func newDaemonSet(customObject v1alpha1.FlannelConfig, etcdEndpoints []string, etcdCAFile, etcdCrtFile, etcdKeyFile string) *appsv1.DaemonSet {
	etcdEndpointsString := strings.Join(etcdEndpoints, ",")

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "daemonset",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.NetworkID,
			Namespace: key.NetworkNamespace(customObject),
			Annotations: map[string]string{
				VersionBundleVersionAnnotation: key.VersionBundleVersion(customObject),
			},
			Labels: map[string]string{
				"app":      key.NetworkID,
				"cluster":  key.ClusterID(customObject),
				"customer": key.ClusterCustomer(customObject),
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":     key.NetworkID,
					"cluster": key.ClusterID(customObject),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
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
							Image:           key.FlannelDockerImage,
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"/bin/sh",
								"-c",
								fmt.Sprintf("/opt/bin/flanneld --etcd-endpoints=%s --etcd-cafile=${ETCD_CA} --etcd-certfile=${ETCD_CRT} --etcd-keyfile=${ETCD_KEY} --etcd-prefix=${ETCD_PREFIX} --iface=${NETWORK_INTERFACE_NAME} --subnet-file=${NETWORK_ENV_FILE_PATH} -v=0", etcdEndpointsString),
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
									Path: "/etc/kubernetes/ssl/etcd",
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
					ServiceAccountName: key.ServiceAccountName(customObject),
				},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: key.MaxUnavailable(),
				},
			},
		},
	}
}
