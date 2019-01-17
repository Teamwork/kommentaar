package path

type queryRef struct {
	ID int64 `query:"id"` // Hello {unknownkeyword}
}

// POST /path
//
// Query: queryRef
// Response 200: {empty}
