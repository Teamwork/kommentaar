package path

// resp docs.
type resp struct {
	other

	Tagged `json:"tag"`
	Basic  string `json:"basic"` // Basic comment.
}

// other docs.
type other struct {
	Other string `json:"other"` // Other.
}

// Tagged docs.
type Tagged struct {
	Tagged string `json:"tagged"` // Tagged.
}

// POST /path
//
// Response 200: $ref: resp
