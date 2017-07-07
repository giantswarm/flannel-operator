package bridge

import (
	"github.com/giantswarm/flanneltpr/docker"
)

// Bridge holds information used for the execution of the Docker entrypoint of
// https://github.com/giantswarm/k8s-network-bridge.
type Bridge struct {
	// Config contains network bridge configuration.
	Config Config `json:"config" yaml:"config"`
	// Docker describes the docker image running k8s-network-bridge.
	Docker docker.Docker `json:"docker" yaml:"docker"`
}
