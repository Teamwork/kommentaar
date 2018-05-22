package params

type pathRef struct{}
type queryRef struct{}
type formRef struct{}

// Make sure that any amount of blank lines doesn't change the output.

// POST /path tag
//
//
// Path: $ref: pathRef
//
//
// Query: $ref: queryRef
//
// Form: $ref: formRef
// Response 200: $empty
//
