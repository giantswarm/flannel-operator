package v2

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "flannel",
				Description: "Flannel version updated.",
				Kind:        "changed",
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "flannel",
				Version: "0.9.0",
			},
		},
		Name:    "flannel-operator",
		Version: "0.1.0",
	}
}
