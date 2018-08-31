package path

type queryRef struct {
	ID int64 `query:"id"` // {readonly}
}

type reqRef struct {
	ID int64 `query:"id"` // {readonly}
}

// POST /path
//
// Query: $ref: queryRef
// Request body: $ref: reqRef
// Response 200: $empty
