package legacy

import (
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newPodAffinity create an affinity ensuring that pod of the app in the
// namespace will not be scheduled on the same node.
func newPodAffinity(spec v1alpha1.FlannelConfigSpec) *apiv1.Affinity {
	return &apiv1.Affinity{
		PodAntiAffinity: &apiv1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []apiv1.PodAffinityTerm{
				{
					LabelSelector: &apismetav1.LabelSelector{
						MatchExpressions: []apismetav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: apismetav1.LabelSelectorOpIn,
								Values:   []string{destroyerApp},
							},
						},
					},
					TopologyKey: "kubernetes.io/hostname",
					Namespaces:  []string{destroyerNamespace(spec)},
				},
			},
		},
	}
}
