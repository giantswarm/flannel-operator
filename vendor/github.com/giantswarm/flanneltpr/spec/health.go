package spec

import "github.com/giantswarm/flanneltpr/spec/health"

type Health struct {
	// Docker describes the docker image running flanneld.
	Docker health.Docker `json:"docker" yaml:"docker"`
}
