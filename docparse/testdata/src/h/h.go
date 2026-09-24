package h

import "repb/report"

// ref holds a field of the Nested type of package repb/report. Package
// repa/report has the same name and also declares a type Nested (see
// testdata/src/g). So the lookup "report.Nested" can give two types.
type ref struct {
	field report.Nested
}
