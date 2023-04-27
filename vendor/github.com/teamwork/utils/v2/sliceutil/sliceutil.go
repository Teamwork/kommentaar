// Package sliceutil provides functions for working with slices.
//
// The "set" helpers are simple implementations, and don't operate on true
// "sets" (e.g. it will retain order, []int64 can contain duplicates). Consider
// using something like golang-set if you want to use sets and care a lot about
// performance: https://github.com/deckarep/golang-set
package sliceutil // import "github.com/teamwork/utils/v2/sliceutil"

import (
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Join converts a slice of T to a comma separated string. Useful for
// inserting into a query without the option of parameterization.
func Join[T any](tt []T) string {
	var str []string
	for _, t := range tt {
		str = append(str, fmt.Sprintf("%v", t))
	}

	return strings.Join(str, ", ")
}

// Unique removes duplicate entries from a list. The list does not have to be sorted.
func Unique[T comparable](tt []T) []T {
	var unique []T
	seen := make(map[T]struct{})
	for _, t := range tt {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			unique = append(unique, t)
		}
	}

	return unique
}

// MergeUnique takes a slice of slices and returns an unsorted slice of
// unique entries.
func MergeUnique[T comparable](tt [][]T) (result []T) {
	var m = make(map[T]bool)

	for _, t := range tt {
		for _, i := range t {
			m[i] = true
		}
	}

	for k := range m {
		result = append(result, k)
	}

	return result
}

// CSVtoInt64Slice converts a string of integers to a slice of int64.
func CSVtoInt64Slice(csv string) ([]int64, error) {
	csv = strings.TrimSpace(csv)
	if len(csv) == 0 {
		return []int64(nil), nil
	}

	items := strings.Split(csv, ",")
	ints := make([]int64, len(items))
	for i, item := range items {
		val, err := strconv.Atoi(strings.TrimSpace(item))
		if err != nil {
			return nil, err
		}
		ints[i] = int64(val)
	}

	return ints, nil
}

// Contains returns true if item is in the provided slice.
func Contains[T comparable](tt []T, item T) bool {
	for _, t := range tt {
		if t == item {
			return true
		}
	}
	return false
}

// InFoldedStringSlice reports whether str is within list(case-insensitive)
func InFoldedStringSlice(list []string, str string) bool {
	for _, item := range list {
		if strings.EqualFold(item, str) {
			return true
		}
	}
	return false
}

// Repeat returns a slice with the item t reated n times.
func Repeat[T any](t T, n int) (tt []T) {
	for i := 0; i < n; i++ {
		tt = append(tt, t)
	}
	return tt
}

// Choose chooses a random item from the list.
func Choose[T any](tt []T) T {
	if len(tt) == 0 {
		var zero T
		return zero
	}
	source := rand.NewSource(time.Now().UnixNano())
	return tt[rand.New(source).Intn(len(tt))]
}

// Range creates an []int counting at "start" up to (and including) "end".
func Range(start, end int) []int {
	rng := make([]int, end-start+1)
	for i := 0; i < len(rng); i++ {
		rng[i] = start + i
	}
	return rng
}

// Remove removes any occurrence of a string from a slice.
func Remove[T comparable](tt []T, remove T) (out []T) {
	for _, t := range tt {
		if t != remove {
			out = append(out, t)
		}
	}

	return out
}

// Filter filters a list. The function will be called for every item and
// those that return false will not be included in the returned list.
func Filter[T comparable](tt []T, fn func(T) bool) []T {
	var ret []T
	for _, t := range tt {
		if fn(t) {
			ret = append(ret, t)
		}
	}

	return ret
}

// FilterEmpty can be used as an argument for Filter() and will
// return false if e is zero value.
func FilterEmpty[T comparable](t T) bool {
	var zero T
	return t != zero
}

// Map returns a list where each item in list has been modified by fn
func Map[T comparable](tt []T, fn func(T) T) []T {
	ret := make([]T, len(tt))
	for i, t := range tt {
		ret[i] = fn(t)
	}

	return ret
}

// InterfaceSliceTo converts []interface to any given slice.
// It will ~optimistically~ try to convert interface item to the dst item type
func InterfaceSliceTo(src []interface{}, dst interface{}) interface{} {
	dstt := reflect.TypeOf(dst)
	if dstt.Kind() != reflect.Slice {
		panic("`dst` is not an slice")
	}

	dstV := reflect.ValueOf(dst)

	for i := range src {
		if i < dstV.Len() {
			dstV.Index(i).Set(reflect.ValueOf(src[i]).Convert(dstt.Elem()))
			continue
		}
		dstV = reflect.Append(dstV, reflect.ValueOf(src[i]).Convert(dstt.Elem()))
	}

	return dstV.Interface()
}
