package req

type reqRef struct {
	Foo string `sometag:"foo"`
}

// POST /path
//
// Request body: reqRef
// Response 200: {empty}
