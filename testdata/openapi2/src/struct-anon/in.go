package path

// response of a pipeline request
type resp struct {
	// pipe it!
	Pipeline struct {
		// Name of the pipeline
		Name  string `json:"name"`
		Foo Foo
	} `json:"pipeline"`
}

type Foo struct {
	bar string
}

// POST /path
//
// Response 200: $ref: resp
