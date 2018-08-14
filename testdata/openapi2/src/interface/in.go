package path

import "interface/otherpkg"

type resp struct {
	Fooer          fooer       `json:"fooer"`
	Fooers         []fooer     `json:"fooers"`
	EmptyInterface interface{} `json:"emptyInterface"`
	OtherPkg       otherpkg.I  `json:"otherPkg"`
}

// fooer is something.
type fooer interface{}

// GET /path
//
// Response 200: $ref: resp
