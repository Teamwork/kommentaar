package path

// response of a pipeline request
type resp struct {
	// pipe it!
	Pipeline struct {
		// Name of the pipeline
		Name  string `json:"name"`
		NoTag int
	} `json:"pipeline"`
}

// POST /path
//
// Response 200: resp
