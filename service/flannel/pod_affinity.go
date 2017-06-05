package flannel

import (
	"github.com/giantswarm/flanneltpr"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/unversioned"
)

// newPodAffinity create an affinity ensuring that pod of the app in the
// namespace will not be scheduled on the same node.
func newPodAffinity(spec flanneltpr.Spec) *api.Affinity {
	return &api.Affinity{
		PodAntiAffinity: &api.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []api.PodAffinityTerm{
				{
					LabelSelector: &unversioned.LabelSelector{
						MatchExpressions: []unversioned.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: unversioned.LabelSelectorOpIn,
								Values:   []string{app(spec)},
							},
						},
					},
					TopologyKey: "kubernetes.io/hostname",
					Namespaces:  []string{namespace},
				},
			},
		},
	}
}
