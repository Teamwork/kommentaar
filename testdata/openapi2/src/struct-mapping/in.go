package path

import "struct-mapping/otherpkg"

type a struct {
	Foo            otherpkg.Foo   `json:"foo"`
	Foos           otherpkg.Foos  `json:"foos"`
	Time           otherpkg.Time  `json:"time"`
	NullableString NullableString `json:"nullableString"`
	State          otherpkg.State `json:"state"`
	StringyInt     StringyInt     `json:"stringyInt"`
}

type StringyInt int

type NullableString struct {
}

// POST /path
//
// Request body: a
// Response 200: a
