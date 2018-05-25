package params

type pathRef struct{ id string }
type queryRef struct{ id string }
type formRef struct{ id string }

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
