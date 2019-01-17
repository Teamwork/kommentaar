package path

import "encoding/json"

// Ideally we don't want dontInclude in the output, as it's not referenced
// anywhere; but that's a bit hard with the current design.

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
// Request body: a
// Response 200: a
