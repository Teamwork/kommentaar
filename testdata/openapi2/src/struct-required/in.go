package path

type queryRef struct {
	ID        int64  `query:"id"`        // {required}
	CompanyID int64  `query:"companyID"` // Hello there {required}
	Ignore    string `query:"-"`
}

// POST /path
//
// Query: $ref: queryRef
// Response 200: $empty
