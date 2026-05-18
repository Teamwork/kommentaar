package docparse

import "net/mail"

// For tests. We don't parse test files.

// testObject general documentation.
//
//nolint:unused
type testObject struct {
	// ID documentation {required}.
	ID int

	// Foo is a really cool foo-thing!
	// Such foo!
	// {optional}
	Foo string

	Bar []string
}

// testEmbedJSONDash covers the case where a struct embeds a qualified
// pointer type purely for Go-level method promotion and excludes it from
// the schema with `json:"-"`. The embed must not be resolved.
//
//nolint:unused
type testEmbedJSONDash struct {
	*mail.Address `json:"-"`

	// ID documentation {required}.
	ID int
}
