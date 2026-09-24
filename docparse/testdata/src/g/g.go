package g

import "repa/report"

// ref holds a field of the Nested type of package repa/report. Package
// repb/report has the same name and also declares a type Nested (see
// testdata/src/h). So the lookup "report.Nested" can give two types.
type ref struct {
	field report.Nested
}
