package req

type reqRef struct {
	Foo string `sometag:"foo"`
}

// POST /path
//
// Request body: $ref: reqRef
// Response 200: $empty
