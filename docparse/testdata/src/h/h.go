package h

import "repb/report"

// ref holds a field of the repb/report package's Nested type. Its package
// name is also "report" (see testdata/src/g), so a lookup that names
// "report.Nested" collides between the two files.
type ref struct {
	field report.Nested
}
