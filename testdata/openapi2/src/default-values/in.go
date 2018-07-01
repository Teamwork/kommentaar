package path

type ref struct {
	ID     int64  `json:"id"`    // {default: 42}
	Field  string `json:"field"` // {default: field value}
	Str    string `json:"str"`   // {default: 666}
	Ignore string `json:"-"`
}

// POST /path
//
// Response 200: $ref: ref
