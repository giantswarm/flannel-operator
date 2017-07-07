package flannel

import (
	"github.com/giantswarm/flanneltpr/docker"
)

// Flannel holds the configuration to run falnneld and create etcd VNI
// configuration for it.
// https://github.com/coreos/flannel/blob/master/Documentation/configuration.md
type Flannel struct {
	// Config contains flannel configuration.
	Config Config `json:"config" yaml:"config"`
	// Docker describes the docker image running flanneld.
	Docker docker.Docker `json:"docker" yaml:"docker"`
}
