package path

import "encoding/json"

// I've got a title already
type dontInclude struct {
	DontIncludeThis string
}

type a struct {
	// {schema: override.yaml}
	Overridden json.RawMessage `json:"overridden"`

	// Got a title already {schema: override.yaml}
	B dontInclude
}

// POST /path
//
// Request body: $ref: a
// Response 200: $ref: a
