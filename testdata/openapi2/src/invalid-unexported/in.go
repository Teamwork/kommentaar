package req

type ref struct {
	Exported    string `query:"exported"`
	notExported string `query:"not"`
}

// POST /path
//
// Query: ref
// Response 200: {empty}
