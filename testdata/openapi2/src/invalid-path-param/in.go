package path

type pathRef struct {
	ID        int64  `path:"id"`
	CompanyID int64  `path:"doesntexist"`
	Ignore    string `path:"-"`
}

// POST /path/{companyID}/{id} tag
//
// Path: $ref: pathRef
// Response 200: $empty
