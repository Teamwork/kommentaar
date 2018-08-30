package params

type pathRef struct {
	ID string `path:"id"`
}
type queryRef struct {
	ID string `query:"id"`
}
type formRef struct {
	ID string `form:"id"`
}

// POST /path/{id} tag
// GET /foo tag
// PUT /bar somethingelse
// Tagline
//
// Desc
//
// Path: $ref: pathRef
// Query: $ref: queryRef
// Form: $ref: formRef
// Response 200: $empty
