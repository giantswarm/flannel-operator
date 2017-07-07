package flannel

type Config struct {
	// Network is the subnet specification, e.g. 10.0.9.0/16.
	Network string `json:"network" yaml:"network"`
	// SubnetLen is the size of the subnet allocated to each host.
	SubnetLen int `json:"subnetLen" yaml:"subnetLen"`
	// VNI is the vxlan network identifier, e.g. 9.
	VNI int `json:"vni" yaml:"vni"`
}
