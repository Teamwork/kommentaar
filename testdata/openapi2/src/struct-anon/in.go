package path

// response of a pipeline request
type resp struct {
	// pipe it!
	Pipeline struct {
		// Name of the pipeline
		Name  string `json:"name"`
		NoTag int
		Named t
	} `json:"pipeline"`
	Named2 t
}

type t struct{ i int }

// POST /path
//
// Response 200: $ref: resp
