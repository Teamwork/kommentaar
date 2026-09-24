package embed_required

// Base is embedded by value.
type Base struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Note     *string `json:"note"`
	Shadowed string  `json:"shadowed"`
	Dup      string  `json:"dup"`
}

// Clash is embedded by value and shares a key with Base.
type Clash struct {
	Dup string `json:"dup"`
}

// Extra is embedded by pointer.
type Extra struct {
	Code string `json:"code"`
}

// resp docs.
type resp struct {
	Base
	Clash
	*Extra

	Shadowed *string `json:"shadowed"`
	Own      string  `json:"own"`
}

// POST /path
//
// Response 200: resp
