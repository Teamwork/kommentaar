package params

type pathRef struct {
	ID string `path:"id"`
}
type queryRef struct {
	// Foo!
	ID string `query:"id"`
}
type formRef struct {
	ID string `form:"id"` // {date-time}
}

// POST /path/{id} tag
//
// Path: pathRef
// Query: queryRef
// Form: formRef
// Response 200: {empty}
