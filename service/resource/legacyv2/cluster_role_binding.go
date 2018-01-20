package legacyv2

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"k8s.io/api/rbac/v1beta1"
	apismeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/flannel-operator/service/keyv2"
)

func newClusterRoleBinding(customObject v1alpha1.FlannelConfig) *v1beta1.ClusterRoleBinding {
	app := networkApp

	clusterRoleBinding := &v1beta1.ClusterRoleBinding{
		TypeMeta: apismeta.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: v1beta1.GroupName,
		},
		ObjectMeta: apismeta.ObjectMeta{
			Name: clusterRoleBinding(customObject.Spec),
			Annotations: map[string]string{
				VersionBundleVersionAnnotation: keyv2.VersionBundleVersion(customObject),
			},
			Labels: map[string]string{
				"app":         app,
				"cluster-id":  clusterName(customObject.Spec),
				"customer-id": clusterCustomer(customObject.Spec),
			},
		},
		Subjects: []v1beta1.Subject{
			{
				Kind:      v1beta1.ServiceAccountKind,
				Namespace: networkNamespace(customObject.Spec),
				Name:      serviceAccountName(customObject.Spec),
			},
		},
		RoleRef: v1beta1.RoleRef{
			APIGroup: v1beta1.GroupName,
			Kind:     "ClusterRole",
			Name:     "flannel-operator",
		},
	}

	return clusterRoleBinding
}

func newClusterRoleBindingPodSecurityPolicy(customObject v1alpha1.FlannelConfig) *v1beta1.ClusterRoleBinding {
	app := networkApp

	clusterRoleBinding := &v1beta1.ClusterRoleBinding{
		TypeMeta: apismeta.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: v1beta1.GroupName,
		},
		ObjectMeta: apismeta.ObjectMeta{
			Name: clusterRoleBindingForPodSecurityPolicy(customObject.Spec),
			Annotations: map[string]string{
				VersionBundleVersionAnnotation: keyv2.VersionBundleVersion(customObject),
			},
			Labels: map[string]string{
				"app":         app,
				"cluster-id":  clusterName(customObject.Spec),
				"customer-id": clusterCustomer(customObject.Spec),
			},
		},
		Subjects: []v1beta1.Subject{
			{
				Kind:      v1beta1.ServiceAccountKind,
				Namespace: networkNamespace(customObject.Spec),
				Name:      serviceAccountNamePodSecurityPolicy(customObject.Spec),
			},
		},
		RoleRef: v1beta1.RoleRef{
			APIGroup: v1beta1.GroupName,
			Kind:     "ClusterRole",
			Name:     "flannel-operator-psp",
		},
	}

	return clusterRoleBinding
}
