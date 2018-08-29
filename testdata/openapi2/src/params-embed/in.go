package params

type common struct {
	// FieldTasks common description.
	//
	// {enum: name priority status description}
	FieldTasks []string `query:"fields[tasks]"`
}

type queryRef struct {
	common

	// Size of page {default: 10, range: 10-100, required}.
	PageSize int64 `query:"pageSize"`
}

// GET /path
//
// Query: $ref: queryRef
// Response 200: $empty
