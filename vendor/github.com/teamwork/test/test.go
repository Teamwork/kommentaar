// Package test contains helper functions that are useful for writing tests.
package test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ErrorContains checks if the error message in out contains the text in
// want.
//
// This is safe when out is nil. Use an empty string for want if you want to
// test that err is nil.
func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

// Read data from a file.
func Read(t *testing.T, paths ...string) []byte {
	t.Helper()

	path := filepath.Join(paths...)
	file, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %v: %v", path, err)
	}
	return file
}

// TempFile creates a new temporary file and returns the path and a clean
// function to remove it.
//
//  f, clean := TempFile("some\ndata")
//  defer clean()
func TempFile(t *testing.T, data string) (string, func()) {
	t.Helper()

	fp, err := ioutil.TempFile(os.TempDir(), "gotest")
	if err != nil {
		t.Fatalf("test.TempFile: could not create file in %v: %v", os.TempDir(), err)
	}

	defer func() {
		err := fp.Close()
		if err != nil {
			t.Fatalf("test.TempFile: close: %v", err)
		}
	}()

	_, err = fp.WriteString(data)
	if err != nil {
		t.Fatalf("test.TempFile: write: %v", err)
	}

	return fp.Name(), func() {
		err := os.Remove(fp.Name())
		if err != nil {
			t.Errorf("test.TempFile: cannot remove %#v: %v", fp.Name(), err)
		}
	}
}

// NormalizeIndent removes tab indentation from every line.
//
// This is useful for "inline" multiline strings:
//
//   cases := []struct {
//       string in
//   }{
//       `
//	 	    Hello,
//	 	    world!
//       `,
//   }
//
// This is nice and readable, but the downside is that every line will now have
// two extra tabs. This will remove those two tabs from every line.
//
// The amount of tabs to remove is based only on the first line, any further
// tabs will be preserved.
func NormalizeIndent(in string) string {
	indent := 0
	for _, c := range strings.TrimLeft(in, "\n") {
		if c != '\t' {
			break
		}
		indent++
	}

	r := ""
	for _, line := range strings.Split(in, "\n") {
		r += strings.Replace(line, "\t", "", indent) + "\n"
	}

	return strings.TrimSpace(r)
}

// R recovers a panic and cals t.Fatal().
//
// This is useful especially in subtests when you want to run a top-level defer.
// Subtests are run in their own goroutine, so those aren't called on regular
// panics. For example:
//
//   func TestX(t *testing.T) {
//       clean := someSetup()
//       defer clean()
//
//       t.Run("sub", func(t *testing.T) {
//           panic("oh noes")
//       })
//   }
//
// The defer is never called here. To fix it, call this function in all
// subtests:
//
//   t.Run("sub", func(t *testing.T) {
//       defer test.R(t)
//       panic("oh noes")
//   })
//
// See: https://github.com/golang/go/issues/20394
func R(t *testing.T) {
	t.Helper()
	r := recover()
	if r != nil {
		t.Fatalf("panic recover: %v", r)
	}
}

// SP makes a new String Pointer.
func SP(s string) *string { return &s }

// I64P makes a new Int64 Pointer.
func I64P(i int64) *int64 { return &i }
