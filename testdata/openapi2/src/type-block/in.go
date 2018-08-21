package req

type (
	foo    struct{}
	reqRef struct{}
)

// POST /path
//
// Request body: $ref: reqRef
// Response 200: $empty
