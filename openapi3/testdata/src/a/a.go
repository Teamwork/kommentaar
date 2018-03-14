// nolint
package a

// GET /
//
// Response: $ref: foo

// doc
type foo struct {
	// Documented str field.
	// Newline.
	str    string
	byt    []byte
	r      rune
	b      bool // Inline docs.
	fl     float64
	err    error
	strP   *string
	slice  []string
	sliceP []*string
	cstr   customStr
	cstrP  *customStr
	bar    bar
	barP   *bar

	// TODO:
	//pkg    mail.Address
	//pkgSlice  []mail.Address
	//pkgSliceP []*mail.Address
	//m      map[string]int
}

type customStr string

// Document me bar!
type bar struct {
	str string
	num uint32 // uint32 docs!
}
