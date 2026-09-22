package a

import (
	"net/mail"

	"b"

	aliased "c"
	other "d/c"
)

// GET /
//
// Response: foo

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
	// {enum}
	enumStr customStr
	// {enum}
	enumsStr  []customStr
	bar       bar
	barP      *bar
	pkg       mail.Address
	pkgSlice  []mail.Address
	pkgSliceP []*mail.Address
	cSlice    []customStr
	deeper    refAnother

	// This has some documentation! {required}
	// {enum: one two three
	//	four five six seven}
	docs string
	// m      map[string]int
}

type nested struct {
	deeper refAnother
}

// mapped exercises map-types resolution against both bare-ident and
// selector references, in both single-field and slice-element form.
type mapped struct {
	b        bar
	bSlice   []bar
	pkg      mail.Address
	pkgSlice []mail.Address
}

type customStrs []customStr

type customStr string

const (
	customStrA customStr = "a"
	customStrB customStr = "b"
	customStrC customStr = "c"
)

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

type withExternalEnum struct {
	// {enum}
	status b.StatusType
	// {enum}
	statuses []b.StatusType
}

// maps exercises resolveMap against primitive, pointer, struct, slice,
// nested map and import alias value types.
type maps struct {
	prim       map[string]int
	primP      map[string]*int
	anyVal     map[string]any
	strct      map[string]bar
	strctP     map[string]*bar
	pkg        map[string]mail.Address
	slice      map[string][]bar
	nested     map[string]map[string]bar
	sliceOfMap map[string][]mail.Address
	// aliasPkg holds a type that the file reaches through an import alias.
	aliasPkg map[string]aliased.Nested
	// aliasPkgSlice holds the same type in a slice.
	aliasPkgSlice map[string][]aliased.Nested
}

// mapsCollide holds two types of the same name from two packages that share a
// base name.
type mapsCollide struct {
	first  map[string]aliased.Nested
	second map[string]other.Nested
}
