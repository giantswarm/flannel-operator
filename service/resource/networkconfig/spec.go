package networkconfig

type Backend struct {
	Type string
	VNI  int
}

type NetworkConfig struct {
	Network   string
	SubnetLen int
	Backend   Backend
}
