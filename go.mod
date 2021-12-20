module github.com/giantswarm/flannel-operator

go 1.14

require (
	github.com/coreos/etcd v3.3.25+incompatible
	github.com/giantswarm/apiextensions v0.4.20
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/k8sclient v0.2.0
	github.com/giantswarm/microendpoint v0.2.0
	github.com/giantswarm/microerror v0.4.0
	github.com/giantswarm/microkit v1.0.0
	github.com/giantswarm/micrologger v0.6.0
	github.com/giantswarm/operatorkit v0.2.1
	github.com/giantswarm/versionbundle v1.0.0
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/viper v1.10.0
	google.golang.org/grpc/examples v0.0.0-20211216223938-51835dc0912b // indirect
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
)

replace (
	github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/etcd v3.3.13+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/etcd v3.3.15+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/etcd v3.3.17+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
)
