package etcd

import "github.com/giantswarm/flannel-operator/flag/service/etcd/tls"

type Etcd struct {
	Endpoint string
	TLS      tls.TLS
}
