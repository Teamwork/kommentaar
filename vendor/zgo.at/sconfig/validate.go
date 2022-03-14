package sconfig

// This file contains some type handlers that can be used for validation.

import (
	"errors"
	"fmt"
)

// Errors used by the validation handlers.
var (
	errValidateNoValue         = errors.New("does not accept any values")
	errValidateSingleValue     = errors.New("must have exactly one value")
	errValidateValueLimitMore  = "must have more than %v values (has: %v)"
	errValidateValueLimitFewer = "must have fewer than %v values (has: %v)"
)

// ValidateNoValue returns a type handler that will return an error if there are
// any values.
func ValidateNoValue() TypeHandler {
	return func(v []string) (interface{}, error) {
		if len(v) != 0 {
			return nil, errValidateNoValue
		}
		return v, nil
	}
}

// ValidateSingleValue returns a type handler that will return an error if there
// is more than one value or if there are no values.
func ValidateSingleValue() TypeHandler {
	return func(v []string) (interface{}, error) {
		if len(v) != 1 {
			return nil, errValidateSingleValue
		}
		return v, nil
	}
}

// ValidateValueLimit returns a type handler that will return an error if there
// either more values than max, or fewer values than min.
func ValidateValueLimit(min, max int) TypeHandler {
	return func(v []string) (interface{}, error) {
		switch {
		case min > 0 && len(v) < min:
			return nil, fmt.Errorf(errValidateValueLimitMore, min, len(v))
		case max > 0 && len(v) > max:
			return nil, fmt.Errorf(errValidateValueLimitFewer, max, len(v))
		default:
			return v, nil
		}
	}
}
