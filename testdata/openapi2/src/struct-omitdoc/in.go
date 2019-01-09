package path

type pathRef struct {
	ID        int64 `path:"id"`
	CompanyID int64 `path:"companyID"` // Hello there {omitdoc}
}
type queryRef struct {
	ID        int64 `query:"id"`
	CompanyID int64 `query:"companyID"` // Hello there {omitdoc}
}

type formRef struct {
	ID        int64 `form:"id"`
	CompanyID int64 `form:"companyID"` // Hello there {omitdoc}
}

type req struct {
	ID        int64 `json:"id"`
	CompanyID int64 `json:"companyID"` // Hello there {omitdoc}
}

type resp struct {
	ID        int64 `json:"id"`
	CompanyID int64 `json:"companyID"` // Hello there {omitdoc}
}

// POST /path/{companyID}/{id}
//
// Path:         $ref: pathRef
// Query:        $ref: queryRef
// Form:         $ref: formRef
// Request body: $ref: req
// Response 200: $ref: resp
