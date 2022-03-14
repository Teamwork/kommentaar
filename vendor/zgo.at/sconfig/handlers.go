package sconfig

import (
	"fmt"
	"strconv"
	"strings"
)

// This file contains the default handler functions for Go's primitives.

func init() {
	defaultTypeHandlers()
}

func defaultTypeHandlers() {
	typeHandlers = map[string][]TypeHandler{
		"string":            {handleString},
		"bool":              {handleBool},
		"float32":           {ValidateSingleValue(), handleFloat32},
		"float64":           {ValidateSingleValue(), handleFloat64},
		"int64":             {ValidateSingleValue(), handleInt64},
		"uint64":            {ValidateSingleValue(), handleUint64},
		"[]string":          {ValidateValueLimit(1, 0), handleStringSlice},
		"[]bool":            {ValidateValueLimit(1, 0), handleBoolSlice},
		"[]float32":         {ValidateValueLimit(1, 0), handleFloat32Slice},
		"[]float64":         {ValidateValueLimit(1, 0), handleFloat64Slice},
		"[]int64":           {ValidateValueLimit(1, 0), handleInt64Slice},
		"[]uint64":          {ValidateValueLimit(1, 0), handleUint64Slice},
		"map[string]string": {ValidateValueLimit(2, 0), handleStringMap},
	}
}

func handleString(v []string) (interface{}, error) {
	return strings.Join(v, " "), nil
}

func handleBool(v []string) (interface{}, error) {
	r, err := parseBool(strings.Join(v, ""))
	if err != nil {
		return nil, err
	}
	return r, nil
}

func parseBool(v string) (bool, error) {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on", "enable", "enabled", "":
		return true, nil
	case "0", "false", "no", "off", "disable", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf(`unable to parse "%s" as a boolean`, v)
	}
}

func handleFloat32(v []string) (interface{}, error) {
	r, err := strconv.ParseFloat(strings.Join(v, ""), 32)
	if err != nil {
		return nil, err
	}
	return float32(r), nil
}
func handleFloat64(v []string) (interface{}, error) {
	r, err := strconv.ParseFloat(strings.Join(v, ""), 64)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func handleInt64(v []string) (interface{}, error) {
	r, err := strconv.ParseInt(strings.Join(v, ""), 10, 64)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func handleUint64(v []string) (interface{}, error) {
	r, err := strconv.ParseUint(strings.Join(v, ""), 10, 64)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func handleStringSlice(v []string) (interface{}, error) {
	return v, nil
}

func handleBoolSlice(v []string) (interface{}, error) {
	a := make([]bool, len(v))
	for i := range v {
		r, err := parseBool(v[i])
		if err != nil {
			return nil, err
		}
		a[i] = r
	}
	return a, nil
}

func handleFloat32Slice(v []string) (interface{}, error) {
	a := make([]float32, len(v))
	for i := range v {
		r, err := strconv.ParseFloat(v[i], 32)
		if err != nil {
			return nil, err
		}
		a[i] = float32(r)
	}
	return a, nil
}

func handleFloat64Slice(v []string) (interface{}, error) {
	a := make([]float64, len(v))
	for i := range v {
		r, err := strconv.ParseFloat(v[i], 64)
		if err != nil {
			return nil, err
		}
		a[i] = r
	}
	return a, nil
}

func handleInt64Slice(v []string) (interface{}, error) {
	a := make([]int64, len(v))
	for i := range v {
		r, err := strconv.ParseInt(v[i], 10, 64)
		if err != nil {
			return nil, err
		}
		a[i] = r
	}
	return a, nil
}

func handleUint64Slice(v []string) (interface{}, error) {
	a := make([]uint64, len(v))
	for i := range v {
		r, err := strconv.ParseUint(v[i], 10, 64)
		if err != nil {
			return nil, err
		}
		a[i] = r
	}
	return a, nil
}

func handleStringMap(v []string) (interface{}, error) {
	if len(v)%2 != 0 {
		return nil, fmt.Errorf("uneven number of arguments: %d", len(v))
	}

	a := make(map[string]string, len(v)/2)
	k := ""
	for i := range v {
		if i%2 == 0 {
			k = v[i]
		} else {
			a[k] = v[i]
		}
	}

	return a, nil
}
