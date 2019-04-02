package etcd

import (
	"github.com/coreos/etcd/client"
	"github.com/giantswarm/microerror"
)

var createFailedError = &microerror.Error{
	Kind: "createFailedError",
}

// IsCreateFailed asserts createFailedError.
func IsCreateFailed(err error) bool {
	return microerror.Cause(err) == createFailedError
}

// IsEtcdKeyAlreadyExists is an error matcher for the v2 etcd client.
func IsEtcdKeyAlreadyExists(err error) bool {
	if cErr, ok := err.(client.Error); ok {
		return cErr.Code == client.ErrorCodeNodeExist
	}
	return false
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var multipleValuesError = &microerror.Error{
	Kind: "multipleValuesError",
}

// IsMultipleValuesFound asserts multipleValuesError.
func IsMultipleValuesFound(err error) bool {
	return microerror.Cause(err) == multipleValuesError
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}
