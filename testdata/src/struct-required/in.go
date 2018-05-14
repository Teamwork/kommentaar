package path

type pathRef struct {
	ID        int64  `path:"id"`        // {required}
	CompanyID int64  `path:"companyID"` // Hello there {required}
	Ignore    string `path:"-"`
}

// POST /path/{companyID}/{id} tag
//
// Path: $ref: pathRef
// Response 200: $empty
