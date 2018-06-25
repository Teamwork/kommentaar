// Package stringutil adds functions for working with strings.
package stringutil // import "github.com/teamwork/utils/stringutil"

import (
	"regexp"
	"strings"
)

// Left returns the "n" left characters of the string.
//
// If the string is shorter than "n" it will return the first "n" characters of
// the string with "…" appended. Otherwise the entire string is returned as-is.
func Left(s string, n int) string {
	if n < 0 {
		n = 0
	}
	if len(s) <= n {
		return s
	}

	var (
		chari int
		bytei int
	)
	for bytei = range s {
		if chari >= n {
			break
		}
		chari++
	}

	return s[:bytei] + "…"
}

// UpperFirst transforms the first character to upper case, leaving the rest of
// the casing alone.
func UpperFirst(s string) string {
	if len(s) < 2 {
		return strings.ToUpper(s)
	}
	for _, c := range s {
		sc := string(c)
		return strings.ToUpper(sc) + s[len(sc):]
	}
	return ""
}

// LowerFirst transforms the first character to lower case, leaving the rest of
// the casing alone.
func LowerFirst(s string) string {
	if len(s) < 2 {
		return strings.ToLower(s)
	}
	for _, c := range s {
		sc := string(c)
		return strings.ToLower(sc) + s[len(sc):]
	}
	return ""
}

var reUnprintable = regexp.MustCompile("[\x00-\x1F\u200e\u200f]")

// RemoveUnprintable removes unprintable characters (0 to 31 ASCII) from a string.
func RemoveUnprintable(s string) string {
	return reUnprintable.ReplaceAllString(s, "")
}

// GetLine gets the nth line \n-denoted line from a string.
func GetLine(in string, n int) string {
	// Would probably be faster to use []byte and find the Nth \n character, but
	// this is "fast enough"™ for now.
	arr := strings.SplitN(in, "\n", n+1)
	if len(arr) <= n-1 {
		return ""
	}
	return arr[n-1]
}
