package path

// resp docs.
type resp struct {
	other
}

// other docs.
type other struct {
	Other string `json:"other"` // Other.
}

// POST /path
//
// Response 200: $ref: resp
// Query: $ref: other
