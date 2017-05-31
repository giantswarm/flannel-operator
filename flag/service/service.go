package service

import (
	"github.com/giantswarm/flannel-operator/flag/service/kubernetes"
)

type Service struct {
	Kubernetes kubernetes.Kubernetes
}
