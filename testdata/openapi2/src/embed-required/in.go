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
	Kind string `json:"kind"` // {required}
}

// Near has x at depth 1.
type Near struct {
	X string `json:"x"`
}

// Far has x at depth 2, so Near wins.
type Far struct {
	Deep
}

// Deep is embedded by Far.
type Deep struct {
	X string `json:"x"`
}

// Tagged sets Bar with a tag, so it wins over Untagged.
type Tagged struct {
	Foo string `json:"Bar"`
}

// Untagged gets Bar from the field name.
type Untagged struct {
	Bar string
}

// Hidden has a key that an {omitdoc} field of resp shadows.
type Hidden struct {
	Secret string `json:"secret"`
}

// Outer is embedded by pointer and embeds Inner by value.
type Outer struct {
	Inner
}

// Inner is embedded by Outer.
type Inner struct {
	Z string `json:"z"` // {required}
	W string `json:"w"`
}

// Wrap is embedded by value and embeds Opt by pointer.
type Wrap struct {
	*Opt
}

// Opt is embedded by Wrap.
type Opt struct {
	V string `json:"v"`
	U string `json:"u"` // {required}
}

// resp docs.
type resp struct {
	Base
	Clash
	*Extra
	Near
	Far
	Tagged
	Untagged
	Hidden
	*Outer
	Wrap

	Shadowed *string `json:"shadowed"`
	Own      string  `json:"own"`
	Secret   string  `json:"secret"` // {omitdoc}
}

// POST /path
//
// Response 200: resp
