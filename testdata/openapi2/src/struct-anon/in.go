package path

type resp struct {
	// pipe it!
	Pipeline struct {
		Name string `json:"name"`
	} `json:"pipeline"`
}

// POST /path
//
// Response 200: $ref: resp
