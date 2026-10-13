package params

type pathRef struct {
	ID string `path:"id"`
}
type queryRef struct {
	// Foo!
	ID string `query:"id"`

	// {schema: filter.yaml}
	Filter string `query:"filter"`
}
type formRef struct {
	ID string `form:"id"` // {date-time} {required}
}

// POST /path/{id} tag
//
// Path: pathRef
// Query: queryRef
// Form: formRef
// Response 200: {empty}
