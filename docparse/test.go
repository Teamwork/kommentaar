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
