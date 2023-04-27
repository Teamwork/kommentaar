// Package diff has some helpers for text diffs.
package diff // import "github.com/teamwork/test/diff"

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/Strum355/go-difflib/difflib"
	"github.com/davecgh/go-spew/spew"
)

// Context indicates the number of lines of context to add.
var Context = 2

// Diff returns a unified diff of the two passed arguments, or "" if they are
// the same.
func Diff(expected, actual interface{}) string {
	if reflect.DeepEqual(expected, actual) {
		return ""
	}
	scs := spew.ConfigState{
		Indent:                  "  ",
		DisableMethods:          true,
		DisablePointerAddresses: true,
		DisableCapacities:       true,
		SortKeys:                true,
	}
	return TextDiff(scs.Sdump(expected), scs.Sdump(actual))
}

// TextDiff returns a unified diff of the two passed strings, or "" if they are
// the same.
func TextDiff(expected, actual string) string {
	return textDiff(expected, actual, false)
}

// TextDiffColored returns a unified diff of the two passed strings, or "" if they are
// the same. This function also colors the output
func TextDiffColored(expected, actual string) string {
	return textDiff(expected, actual, true)
}

func textDiff(expected, actual string, colored bool) string {
	udiff := difflib.UnifiedDiff{
		A:        strings.SplitAfter(expected, "\n"),
		FromFile: "expected",
		B:        strings.SplitAfter(actual, "\n"),
		ToFile:   "actual",
		Context:  Context,
		Colored:  colored,
	}
	diff, err := difflib.GetUnifiedDiffString(udiff)
	if err != nil {
		panic(fmt.Sprintf("Error producing diff: %s\n", err))
	}

	if diff != "" {
		diff = "\n" + diff
	}
	return diff
}

// ContextDiff returns a "context diff" of the two passed strings, or "" if they
// are the same.
//
// This usually works better than a unified diff if the strings are long.
func ContextDiff(expected, actual string) string {
	return contextDiff(expected, actual, false)
}

// ContextDiffColored returns a "context diff" of the two passed strings, or "" if they
// are the same. This function also colors the output
//
// This usually works better than a unified diff if the strings are long.
func ContextDiffColored(expected, actual string) string {
	return contextDiff(expected, actual, true)
}

func contextDiff(expected, actual string, colored bool) string {
	cdiff := difflib.ContextDiff{
		A:        strings.SplitAfter(expected, "\n"),
		FromFile: "expected",
		B:        strings.SplitAfter(actual, "\n"),
		ToFile:   "actual",
		Context:  Context,
		Colored:  colored,
	}
	diff, err := difflib.GetContextDiffString(cdiff)
	if err != nil {
		panic(fmt.Sprintf("Error producing diff: %s\n", err))
	}

	if diff != "" {
		diff = "\n" + diff
	}
	return diff
}

// Cmp returns both the actual and the expected value neatly aligned.
func Cmp(expected, actual interface{}) string {
	return fmt.Sprintf("\nexpected: %#v\nactual:   %#v\n", expected, actual)
}

// JSONDiff does a diff of two JSON strings. It first unmarshals the two JSON
// objects into interface{}s, then does a reflect.DeepEqual(). If there is
// a difference, it then re-marshals the objects into pretty JSON and returns
// a diff. Errors panic.
func JSONDiff(expected, actual []byte) string {
	var e, a interface{}
	if err := json.Unmarshal(expected, &e); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(actual, &a); err != nil {
		panic(err)
	}
	if reflect.DeepEqual(e, a) {
		return ""
	}
	prettyExpected, err := json.MarshalIndent(e, "", "    ")

	if err != nil {
		panic(err)
	}
	prettyActual, err := json.MarshalIndent(a, "", "    ")

	if err != nil {
		panic(err)
	}
	return TextDiff(string(prettyExpected)+"\n", string(prettyActual)+"\n")
}

// MarshalJSONDiff works the same as JSONDiff, but first marshals the two
// objects to JSON.
func MarshalJSONDiff(expected, actual interface{}) string {
	e, err := json.Marshal(expected)
	if err != nil {
		panic(err)
	}
	a, err := json.Marshal(actual)
	if err != nil {
		panic(err)
	}
	return JSONDiff(e, a)
}
