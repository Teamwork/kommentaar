package enumindicators

type query struct {
	// Op is a comparison operator.
	//
	// {enum: = != << IN}
	Op string `query:"op"`
}

// GET /path
//
// Query: query
// Response 200: {empty}
