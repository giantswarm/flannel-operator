package v3

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
				Version: "0.10.0",
			},
		},
		Name:    "flannel-operator",
		Version: "0.2.0",
	}
}
