package project

var (
	description = "The flannel-operator handles flannel for Kubernetes clusters running on Giantnetes."
	gitSHA      = "n/a"
	name        = "flannel-operator"
	source      = "https://github.com/giantswarm/flannel-operator"
	version     = "1.2.0"
)

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
