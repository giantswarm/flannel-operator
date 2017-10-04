package etcdstorage

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = microerror.New("invalid config")

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var multipleValuesError = microerror.New("multiple values")

// IsMultipleValuesFound asserts multipleValuesError.
func IsMultipleValuesFound(err error) bool {
	return microerror.Cause(err) == multipleValuesError
}
