package flanneltpr

import (
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/flanneltpr/spec"
)

type Spec struct {
	Bridge        spec.Bridge          `json:"bridge" yaml:"bridge"`
	Cluster       spec.Cluster         `json:"cluster" yaml:"cluster"`
	Health        spec.Health          `json:"health" yaml:"health"`
	Flannel       spec.Flannel         `json:"flannel" yaml:"flannel"`
	VersionBundle versionbundle.Bundle `json:"version_bundle" yaml:"version_bundle"`
}
