package embed_required

// Base is embedded by value.
type Base struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Note     *string `json:"note"`
	Shadowed string  `json:"shadowed"`
}

// Extra is embedded by pointer.
type Extra struct {
	Code string `json:"code"`
}

// resp docs.
type resp struct {
	Base
	*Extra

	Shadowed *string `json:"shadowed"`
	Own      string  `json:"own"`
}

// POST /path
//
// Response 200: resp
