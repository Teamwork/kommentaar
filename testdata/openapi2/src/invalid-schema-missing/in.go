package path

type req struct {
	// {required} {schema: missing.yaml}
	Child string `json:"child"`
}

// POST /path
//
// Request body: req
// Response 200: {empty}
