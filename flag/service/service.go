package service

import (
	"github.com/giantswarm/operatorkit/flag/service/kubernetes"

	"github.com/giantswarm/flannel-operator/flag/service/crd"
	"github.com/giantswarm/flannel-operator/flag/service/etcd"
)

type Service struct {
	CRD        crd.CRD
	Etcd       etcd.Etcd
	Kubernetes kubernetes.Kubernetes
}
