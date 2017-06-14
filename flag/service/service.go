package service

import (
	"github.com/giantswarm/flannel-operator/flag/service/etcd"
	"github.com/giantswarm/flannel-operator/flag/service/kubernetes"
)

type Service struct {
	Etcd       etcd.Etcd
	Kubernetes kubernetes.Kubernetes
}
