package path

type wrapper[T any] struct {
	Data  T      `json:"data"` // {required}
	Count int    `json:"count"`
	Note  string `json:"note"` // {required}
}

type reqRef struct {
	Wrapped wrapper[string] `json:"wrapped"` // {required}
}

// POST /path
//
// Request body: reqRef
// Response 200: {empty}
