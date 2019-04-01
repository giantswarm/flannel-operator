package etcd

import "github.com/giantswarm/flannel-operator/flag/service/etcd/tls"

type Etcd struct {
	Endpoints string
	TLS       tls.TLS
}
