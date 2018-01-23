package legacyv2

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"k8s.io/api/rbac/v1beta1"
	apismeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config represents the configuration used to create a new cluster role binding resource.
type ClusterRoleBindingConfig struct {
	name             string
	subjectName      string
	subjectNamespace string
	roleName         string
}

// DefaultConfig provides a default configuration to create a new cluster role binding
// resource by best effort.
func ClusterRoleBindingConfigDefaultConfig() ClusterRoleBindingConfig {
	return ClusterRoleBindingConfig{
		name:             "",
		subjectName:      "",
		subjectNamespace: "",
		roleName:         "",
	}
}

func newClusterRoleBinding(customObject v1alpha1.FlannelConfig) *v1beta1.ClusterRoleBinding {
	config := ClusterRoleBindingConfigDefaultConfig()
	config.name = clusterRoleBinding(customObject.Spec)
	config.subjectName = serviceAccountName(customObject.Spec)
	config.subjectNamespace = networkNamespace(customObject.Spec)
	config.roleName = "flannel-operator"
	clusterRoleBinding := createClusterRoleBinding(customObject, config)

	return clusterRoleBinding
}

func newClusterRoleBindingPodSecurityPolicy(customObject v1alpha1.FlannelConfig) *v1beta1.ClusterRoleBinding {
	config := ClusterRoleBindingConfigDefaultConfig()
	config.name = clusterRoleBindingForPodSecurityPolicy(customObject.Spec)
	config.subjectName = serviceAccountNamePodSecurityPolicy(customObject.Spec)
	config.subjectNamespace = networkNamespace(customObject.Spec)
	config.roleName = "flannel-operator-psp"
	clusterRoleBinding := createClusterRoleBinding(customObject, config)

	return clusterRoleBinding
}

func createClusterRoleBinding(customObject v1alpha1.FlannelConfig, config ClusterRoleBindingConfig) *v1beta1.ClusterRoleBinding {
	app := networkApp

	clusterRoleBinding := &v1beta1.ClusterRoleBinding{
		TypeMeta: apismeta.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: v1beta1.GroupName,
		},
		ObjectMeta: apismeta.ObjectMeta{
			Name: config.name,
			Labels: map[string]string{
				"app":         app,
				"cluster-id":  clusterName(customObject.Spec),
				"customer-id": clusterCustomer(customObject.Spec),
			},
		},
		Subjects: []v1beta1.Subject{
			{
				Kind:      v1beta1.ServiceAccountKind,
				Namespace: config.subjectNamespace,
				Name:      config.subjectName,
			},
		},
		RoleRef: v1beta1.RoleRef{
			APIGroup: v1beta1.GroupName,
			Kind:     "ClusterRole",
			Name:     config.roleName,
		},
	}

	return clusterRoleBinding
}
