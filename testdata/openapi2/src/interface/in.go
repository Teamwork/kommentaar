package path

import (
	"interface/otherpkg"
	"io"
)

type resp struct {
	Fooer          fooer       `json:"fooer"`
	Fooers         []fooer     `json:"fooers"`
	EmptyInterface interface{} `json:"emptyInterface"`
	OtherPkg       otherpkg.I  `json:"otherPkg"`

	io.Reader // ensure embedded interface doesn't error
}

// fooer is something.
type fooer interface{}

// GET /path
//
// Response 200: $ref: resp
