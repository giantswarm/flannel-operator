package legacy

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	PrometheusNamespace            = "flannel_operator"
	PrometheusSubsystem            = "flanneld_resource"
	VersionBundleVersionAnnotation = "giantswarm.io/version-bundle-version"
)

var versionBundleVersionGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: PrometheusNamespace,
		Subsystem: PrometheusSubsystem,
		Name:      "version_bundle_version_total",
		Help:      "A metric labeled by major, minor and patch version of the version bundle being in use.",
	},
	[]string{"major", "minor", "patch"},
)
