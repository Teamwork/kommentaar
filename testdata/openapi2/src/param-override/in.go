package path

import "encoding/json"

// I've got a title already
type b struct {
	// Hello
	A string
	B string
	C string
}

type a struct {
	// {override: override.yaml}
	Overridden json.RawMessage `json:"overridden"`
	// Got a title already {override: override.yaml}
	B b
}

// POST /path
//
// Request body: $ref: a
// Response 200: $ref: a
