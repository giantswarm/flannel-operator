package flannel

import (
	"github.com/giantswarm/flanneltpr"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api"
)

// newPodAffinity create an affinity ensuring that pod of the app in the
// namespace will not be scheduled on the same node.
func newPodAffinity(spec flanneltpr.Spec) *api.Affinity {
	return &api.Affinity{
		PodAntiAffinity: &api.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []api.PodAffinityTerm{
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
