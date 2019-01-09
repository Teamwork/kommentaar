package path

import "struct-mapping/otherpkg"

type a struct {
	Foo            otherpkg.Foo   `json:"foo"`
	Foos           otherpkg.Foos  `json:"foos"`
	Time           otherpkg.Time  `json:"time"`
	NullableString NullableString `json:"nullableString"`
}

type NullableString struct {
}

// POST /path
//
// Request body: $ref: a
// Response 200: $ref: a
