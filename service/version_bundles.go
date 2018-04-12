package service

import (
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/flannel-operator/service/controller/v2"
	"github.com/giantswarm/flannel-operator/service/controller/v3"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v2.VersionBundle())
	versionBundles = append(versionBundles, v3.VersionBundle())

	return versionBundles
}
