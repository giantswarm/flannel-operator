package service

import (
	"github.com/giantswarm/versionbundle"

	v3 "github.com/giantswarm/flannel-operator/service/controller/v3"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v3.VersionBundle())

	return versionBundles
}
