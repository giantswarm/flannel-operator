[![CircleCI](https://dl.circleci.com/status-badge/img/gh/giantswarm/flannel-operator/tree/master.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/giantswarm/flannel-operator/tree/master)

# flannel-operator

The flannel-operator handles flannel setup for Kubernetes clusters running on our on-prem environment. Since the on-prem solution works based on Kubernetes Inception, the workload cluster machines are KVM host running inside the management cluster machines as pods. In order to make connectivity possible between workload clusters nodes (running as pods) this operator configures a flannel network overlay that manage connection between end user workloads.

## Getting the Project

Download the latest release:
https://github.com/giantswarm/flannel-operator/releases/latest

Clone the git repository: https://github.com/giantswarm/flannel-operator.git

Download the latest docker image from here:
https://quay.io/repository/giantswarm/flannel-operator


### How to build

```
go build github.com/giantswarm/flannel-operator
```

## Contact

- Mailing list: [giantswarm](https://groups.google.com/forum/!forum/giantswarm)
- Bugs: [issues](https://github.com/giantswarm/flannel-operator/issues)

## Contributing & Reporting Bugs

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches, the
contribution workflow as well as reporting bugs.

For security issues, please see [the security policy](SECURITY.md).


## License

flannel-operator is under the Apache 2.0 license. See the [LICENSE](LICENSE) file
for details.


## Credit
- https://golang.org
