package path

type req struct {
	// {required} {schema: child.txt}
	Child string `json:"child"`
}

// POST /path
//
// Request body: req
// Response 200: {empty}
