package params

type pathRef struct{}
type queryRef struct{}
type formRef struct{}

// POST /path/{id} tag
//
// Path: $ref: pathRef
// Query: $ref: queryRef
// Form: $ref: formRef
// Response 200: $empty
