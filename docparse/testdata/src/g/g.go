package g

import "repa/report"

// ref holds a field of package repa/report's Nested type. Package
// repb/report also declares a type named report.Nested (see
// testdata/src/h), so a lookup that names "report.Nested" collides between
// the two files.
type ref struct {
	field report.Nested
}
