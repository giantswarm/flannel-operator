package service

import (
	"github.com/giantswarm/flannel-operator/flag/service/crd"
	"github.com/giantswarm/flannel-operator/flag/service/etcd"
	"github.com/giantswarm/flannel-operator/flag/service/kubernetes"
)

type Service struct {
	CRD        crd.CRD
	Etcd       etcd.Etcd
	Kubernetes kubernetes.Kubernetes
}
