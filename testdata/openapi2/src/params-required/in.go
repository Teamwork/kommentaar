package path

type pathRef struct {
	ID int64 `path:"id"` // {required}
}

type queryRef struct {
	// {schema: filter.yaml}
	Filter string `query:"filter"`

	// {required} {schema: filter.yaml}
	Sort string `query:"sort"`
}

type formRef struct {
	Name  string `form:"name"` // {required}
	Notes string `form:"notes"`
}

// POST /path/{id}
//
// Path: pathRef
// Query: queryRef
// Form: formRef
// Response 200: {empty}
