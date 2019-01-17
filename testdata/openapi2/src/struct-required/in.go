package path

type queryRef struct {
	ID        int64  `query:"id"`        // {required}
	CompanyID int64  `query:"companyID"` // Hello there {required}
	Ignore    string `query:"-"`
}

type req struct {
	Data struct {
		Meta struct {
			Booly   bool   `json:"booly"` // Another level {required}
			Stringy string `json:"stringy"`
		} `json:"meta"` // {required}
	} `json:"data"` // {required}
	CreatedBy int64 `json:"createdBy"` // {required}
}

// POST /path
//
// Query: $ref: queryRef
// Request body: $ref: req
// Response 200: $empty
