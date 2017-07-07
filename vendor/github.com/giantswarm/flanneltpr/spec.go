package flanneltpr

import (
	"github.com/giantswarm/flanneltpr/bridge"
	"github.com/giantswarm/flanneltpr/cluster"
	"github.com/giantswarm/flanneltpr/flannel"
)

type Spec struct {
	Cluster cluster.Cluster `json:"cluster" yaml:"cluster"`

	Bridge  bridge.Bridge   `json:"bridge" yaml:"bridge"`
	Flannel flannel.Flannel `json:"flannel" yaml:"flannel"`
}
