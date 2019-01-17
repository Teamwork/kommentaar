package req

type (
	foo    struct{}
	reqRef struct{}
)

// POST /path
//
// Request body: reqRef
// Response 200: {empty}
