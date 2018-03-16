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
		Time:         time.Date(2018, time.March, 16, 9, 15, 0, 0, time.UTC),
		Version:      "0.2.0",
		WIP:          false,
	}
}
