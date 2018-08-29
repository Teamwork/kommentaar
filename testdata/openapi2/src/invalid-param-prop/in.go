package path

type queryRef struct {
	ID int64 `query:"id"` // Hello {unknownkeyword}
}

// POST /path
//
// Query: $ref: queryRef
// Response 200: $empty
