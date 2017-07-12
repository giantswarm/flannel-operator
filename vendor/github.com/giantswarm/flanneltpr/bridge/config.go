package bridge

type Config struct {
	// Interface is the network interface name, e.g. bond0.3, or ens33.
	Interface string `json:"interface" yaml:"interface"`
	// PrivateNetwork is the host's private network to block against, e.g.
	// 10.0.4.0/24.
	PrivateNetwork string `json:"privateNetwork" yaml:"privateNetwork"`

	// DNS holds DNS configuration for the bridge.
	DNS DNS `json:"dns" yaml:"dns"`
	// NTP holds NTP configuration for the bridge.
	NTP NTP `json:"ntp" yaml:"ntp"`
}
