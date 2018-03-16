package service

import (
	"github.com/giantswarm/kvm-operator/service/kvmconfig/v3"
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/flannel-operator/service/flannelconfig/v2"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v2.VersionBundle())
	versionBundles = append(versionBundles, v3.VersionBundle())

	return versionBundles
}
