// nolint
package a

import "net/mail"

// GET /
//
// Response: $ref: foo

// doc
type foo struct {
	// Documented str field.
	// Newline.
	str       string
	byt       []byte
	r         rune
	b         bool // Inline docs.
	fl        float64
	err       error
	strP      *string
	slice     []string
	sliceP    []*string
	cstr      customStr
	cstrP     *customStr
	bar       bar
	barP      *bar
	pkg       mail.Address
	pkgSlice  []mail.Address
	pkgSliceP []*mail.Address
	deeper    refAnother

	// This has some documentation! {required}
	// {enum: one two three}
	docs string
	//m      map[string]int
}

type nested struct {
	deeper refAnother
}

type customStr string

// Document me bar!
type bar struct {
	str string
	num uint32 // uint32 docs!
}

type refAnother struct {
	ref refAnother2
}

type refAnother2 struct {
	str   customStr
	strct bar
	pkg   mail.Address
}
