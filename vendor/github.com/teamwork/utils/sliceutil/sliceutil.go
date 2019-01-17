// Package sliceutil provides functions for working with slices.
//
// The "set" helpers are simple implementations, and don't operate on true
// "sets" (e.g. it will retain order, []int64 can contain duplicates). Consider
// using something like golang-set if you want to use sets and care a lot about
// performance: https://github.com/deckarep/golang-set
package sliceutil // import "github.com/teamwork/utils/sliceutil"

import (
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

// JoinInt converts a slice of ints to a comma separated string. Useful for
// inserting into a query without the option of parameterization.
func JoinInt(ints []int64) string {
	var intStr []string
	for _, e := range ints {
		intStr = append(intStr, strconv.Itoa(int(e)))
	}

	return strings.Join(intStr, ", ")
}

// UniqInt64 removes duplicate entries from list. The list does not have to be
// sorted.
func UniqInt64(list []int64) []int64 {
	var unique []int64
	seen := make(map[int64]struct{})
	for _, l := range list {
		if _, ok := seen[l]; !ok {
			seen[l] = struct{}{}
			unique = append(unique, l)
		}
	}
	return unique
}

// UniqString removes duplicate entries from list.
func UniqString(list []string) []string {
	sort.Strings(list)
	var last string
	l := list[:0]
	for _, str := range list {
		if str != last {
			l = append(l, str)
		}
		last = str
	}
	return l
}

// UniqueMergeSlices takes a slice of slices of int64s and returns an unsorted
// slice of unique int64s.
func UniqueMergeSlices(s [][]int64) (result []int64) {
	var m = make(map[int64]bool)

	for _, el := range s {
		for _, i := range el {
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

// InStringSlice reports whether str is within list
func InStringSlice(list []string, str string) bool {
	for _, item := range list {
		if item == str {
			return true
		}
	}
	return false
}

// InIntSlice reports whether i is within list
func InIntSlice(list []int, i int) bool {
	for _, item := range list {
		if item == i {
			return true
		}
	}
	return false
}

// InInt64Slice reports whether i is within list
func InInt64Slice(list []int64, i int64) bool {
	for _, item := range list {
		if item == i {
			return true
		}
	}
	return false
}

// RepeatString returns a slice with the string s reated n times.
func RepeatString(s string, n int) (r []string) {
	for i := 0; i < n; i++ {
		r = append(r, s)
	}
	return r
}

// ChooseString chooses a random item from the list.
func ChooseString(l []string) string {
	if len(l) == 0 {
		return ""
	}
	rand.Seed(time.Now().UnixNano())
	return l[rand.Intn(len(l))]
}

// Range creates an []int counting at "start" up to (and including) "end".
func Range(start, end int) []int {
	rng := make([]int, end-start+1)
	for i := 0; i < len(rng); i++ {
		rng[i] = start + i
	}
	return rng
}

// FilterString filters a list. The function will be called for every item and
// those that return false will not be included in the return value.
func FilterString(list []string, fun func(string) bool) []string {
	var ret []string
	for _, e := range list {
		if fun(e) {
			ret = append(ret, e)
		}
	}

	return ret
}

// FilterStringEmpty can be used as an argument for FilterString() and will
// return false if e is empty or contains only whitespace.
func FilterStringEmpty(e string) bool {
	return strings.TrimSpace(e) != ""
}

// FilterInt filters a list. The function will be called for every item and
// those that return false will not be included in the return value.
func FilterInt(list []int64, fun func(int64) bool) []int64 {
	var ret []int64
	for _, e := range list {
		if fun(e) {
			ret = append(ret, e)
		}
	}

	return ret
}

// FilterIntEmpty can be used as an argument for FilterInt() and will
// return false if e is empty or contains only whitespace.
func FilterIntEmpty(e int64) bool {
	return e != 0
}
