package params

type customType int64

type magic interface{}

type magics []magic

type common struct {
	// FieldTasks common description.
	//
	// {enum: name priority status description}
	FieldTasks []string `query:"fields[tasks]"`

	// CustomTypes common description.
	//
	// {enum: blue red yellow}
	CustomTypes []customType `query:"customTypes"`

	// Magics common description.
	Magics magics `query:"magics"`
}

type queryRef struct {
	common

	// Size of page {default: 10, range: 10-100, required}.
	PageSize int64 `query:"pageSize"`
}

// GET /path
//
// Query: queryRef
// Response 200: {empty}
