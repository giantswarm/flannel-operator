package v3

import (
	"time"

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
		Dependencies: []versionbundle.Dependency{},
		Deprecated:   false,
		Name:         "flannel-operator",
		Time:         time.Date(2017, time.October, 27, 16, 21, 0, 0, time.UTC),
		Version:      "0.1.0",
		WIP:          false,
	}
}
