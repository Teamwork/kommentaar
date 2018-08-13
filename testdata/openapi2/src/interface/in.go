package path

type resp struct {
	Fooer          fooer       `json:"fooer"`
	Fooers         []fooer     `json:"fooers"`
	EmptyInterface interface{} `json:"emptyInterface"`
}

// fooer is something.
type fooer interface{}

// GET /path
//
// Response 200: $ref: resp
