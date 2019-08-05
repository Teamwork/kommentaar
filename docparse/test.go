// nolint
package docparse

// For tests. We don't parse test files.

// testObject general documentation.
type testObject struct {
	// ID documentation {required}.
	ID int

	// Foo is a really cool foo-thing!
	// Such foo!
	// {optional}
	Foo string

	Bar []string
}

func (t testObject) FilterFieldMap() map[string]string {
	return map[string]string{
		"a": "b",
		"c": "d",
	}
}
func (t testObject) SortFieldMap() map[string]string {
	return map[string]string{
		"e": "f",
		"g": "h",
	}
}
