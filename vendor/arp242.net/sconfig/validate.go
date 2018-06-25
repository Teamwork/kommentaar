// Copyright © 2016-2017 Martin Tournoij
// See the bottom of this file for the full copyright.

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

// The MIT License (MIT)
//
// Copyright © 2016-2017 Martin Tournoij
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// The software is provided "as is", without warranty of any kind, express or
// implied, including but not limited to the warranties of merchantability,
// fitness for a particular purpose and noninfringement. In no event shall the
// authors or copyright holders be liable for any claim, damages or other
// liability, whether in an action of contract, tort or otherwise, arising
// from, out of or in connection with the software or the use or other dealings
// in the software.
