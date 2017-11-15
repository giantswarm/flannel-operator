package flanneltpr

import (
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/giantswarm/flanneltpr/spec"
	bridge "github.com/giantswarm/flanneltpr/spec/bridge"
	bridgespec "github.com/giantswarm/flanneltpr/spec/bridge/spec"
	"github.com/giantswarm/flanneltpr/spec/flannel"
	"github.com/giantswarm/flanneltpr/spec/health"
	"github.com/giantswarm/versionbundle"
	"github.com/kylelemons/godebug/pretty"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestSpecYamlEncoding(t *testing.T) {

	spec := Spec{
		Cluster: spec.Cluster{
			Customer:  "batman",
			ID:        "85f2g",
			Namespace: "85f2g",
		},
		Bridge: spec.Bridge{
			Spec: bridge.Spec{
				Interface:      "ens4f1",
				PrivateNetwork: "10.4.10.0/24",
				DNS: bridgespec.DNS{
					Servers: []net.IP{
						net.ParseIP("10.1.101.1"),
						net.ParseIP("10.1.101.2"),
					},
				},
				NTP: bridgespec.NTP{
					Servers: []string{
						"10.1.101.1",
						"10.1.101.2",
					},
				},
			},
			Docker: bridge.Docker{
				Image: "quay.io/giantswarm/k8s-network-bridge",
			},
		},
		Flannel: spec.Flannel{
			Spec: flannel.Spec{
				Network:   "172.26.0.0/16",
				RunDir:    "/run/flannel",
				SubnetLen: 30,
				VNI:       26,
			},
		},
		Health: spec.Health{
			Docker: health.Docker{
				Image: "quay.io/giantswarm/flannel-network-health",
			},
		},
		VersionBundle: versionbundle.Bundle{
			Changelogs: []versionbundle.Changelog{
				{
					Component:   "calico",
					Description: "Calico version updated.",
					Kind:        "changed",
				},
			},
			Components: []versionbundle.Component{
				{
					Name:    "calico",
					Version: "1.1.0",
				},
				{
					Name:    "kube-dns",
					Version: "1.0.0",
				},
			},
			Dependencies: []versionbundle.Dependency{
				{
					Name:    "kubernetes",
					Version: "<= 1.7.x",
				},
			},
			Deprecated: false,
			Name:       "kubernetes-operator",
			Time:       time.Unix(10, 5).In(time.UTC),
			Version:    "0.1.0",
			WIP:        false,
		},
	}

	var got map[string]interface{}
	{
		bytes, err := yaml.Marshal(&spec)
		require.NoError(t, err, "marshaling spec")
		err = yaml.Unmarshal(bytes, &got)
		require.NoError(t, err, "unmarshaling spec to map")
	}

	var want map[string]interface{}
	{
		bytes, err := ioutil.ReadFile("testdata/spec.yaml")
		require.NoError(t, err)
		err = yaml.Unmarshal(bytes, &want)
		require.NoError(t, err, "unmarshaling fixture to map")
	}

	diff := pretty.Compare(want, got)
	require.Equal(t, "", diff, "diff: (-want +got)\n%s", diff)
}
