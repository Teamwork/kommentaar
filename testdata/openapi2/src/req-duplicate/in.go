package req

type reqRef struct{}

// POST /path
//
// Request body: $ref: reqRef
// Request body: $ref: reqRef
// Response 200: $empty
