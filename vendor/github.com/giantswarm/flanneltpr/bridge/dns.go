package bridge

import "net"

type DNS struct {
	Servers []net.IP `json:"servers" yaml:"servers"`
}
