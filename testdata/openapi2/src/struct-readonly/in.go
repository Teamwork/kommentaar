package path

type queryRef struct {
	ID int64 `query:"id"` // {readonly}
}

type reqRef struct {
	ID int64 `query:"id"` // {readonly}
}

// POST /path
//
// Query: queryRef
// Request body: reqRef
// Response 200: {empty}
