package embed_required_explicit

// Base is embedded by value.
type Base struct {
	ID   int64  `json:"id"`
	Name string `json:"name"` // {required}
}

// Extra is embedded by pointer.
type Extra struct {
	Kind string `json:"kind"` // {required}
}

// resp docs.
type resp struct {
	Base
	*Extra

	Own string `json:"own"`
}

// POST /path
//
// Response 200: resp
