package legacy

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"k8s.io/api/rbac/v1beta1"
	apismeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/flannel-operator/service/controller/v3/key"
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
	config.subjectNamespace = key.NetworkNamespace(customObject)
	config.roleName = "flannel-operator"
	clusterRoleBinding := createClusterRoleBinding(customObject, config)

	return clusterRoleBinding
}

func newClusterRoleBindingForDeletion(customObject v1alpha1.FlannelConfig) *v1beta1.ClusterRoleBinding {
	config := ClusterRoleBindingConfigDefaultConfig()
	config.name = clusterRoleBindingForDeletion(customObject.Spec)
	config.subjectName = serviceAccountName(customObject.Spec)
	config.subjectNamespace = destroyerNamespace(customObject.Spec)
	config.roleName = "flannel-operator"
	clusterRoleBinding := createClusterRoleBinding(customObject, config)

	return clusterRoleBinding
}

func newClusterRoleBindingPodSecurityPolicy(customObject v1alpha1.FlannelConfig) *v1beta1.ClusterRoleBinding {
	config := ClusterRoleBindingConfigDefaultConfig()
	config.name = clusterRoleBindingForPodSecurityPolicy(customObject.Spec)
	config.subjectName = serviceAccountName(customObject.Spec)
	config.subjectNamespace = key.NetworkNamespace(customObject)
	config.roleName = "flannel-operator-psp"
	clusterRoleBinding := createClusterRoleBinding(customObject, config)

	return clusterRoleBinding
}

func newClusterRoleBindingPodSecurityPolicyForDeletion(customObject v1alpha1.FlannelConfig) *v1beta1.ClusterRoleBinding {
	config := ClusterRoleBindingConfigDefaultConfig()
	config.name = clusterRoleBindingForPodSecurityPolicyForDeletion(customObject.Spec)
	config.subjectName = serviceAccountName(customObject.Spec)
	config.subjectNamespace = destroyerNamespace(customObject.Spec)
	config.roleName = "flannel-operator-psp"
	clusterRoleBinding := createClusterRoleBinding(customObject, config)

	return clusterRoleBinding
}

func createClusterRoleBinding(customObject v1alpha1.FlannelConfig, config ClusterRoleBindingConfig) *v1beta1.ClusterRoleBinding {
	clusterRoleBinding := &v1beta1.ClusterRoleBinding{
		TypeMeta: apismeta.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: v1beta1.GroupName,
		},
		ObjectMeta: apismeta.ObjectMeta{
			Name: config.name,
			Labels: map[string]string{
				"app":                        networkApp,
				"giantswarm.io/cluster":      key.ClusterID(customObject),
				"giantswarm.io/organization": key.ClusterCustomer(customObject),
				// TODO remove deprecated labels.
				//
				//     https://github.com/giantswarm/giantswarm/issues/5860
				//
				"cluster-id":  key.ClusterID(customObject),
				"customer-id": key.ClusterCustomer(customObject),
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
