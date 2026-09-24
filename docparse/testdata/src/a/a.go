package a

import (
	"net/mail"

	"b"
	"example.com/m"

	aliased "c"
	other "d/c"

	repa "repa/report"
	repb "repb/report"
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

	dotted          m.Items
	dottedSlice     []m.Items
	namedSlice      []bars
	aliasNamedSlice []aliased.Nesteds

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

type bars []bar

// maps exercises resolveMap against primitive, pointer, struct, slice, named
// slice, nested map, import alias and dotted import path value types.
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
	aliasPkgSlice   map[string][]aliased.Nested
	namedSlice      map[string]bars
	aliasNamedSlice map[string]aliased.Nesteds
	dotted          map[string]m.Item
	dottedSlice     map[string]m.Items
	selfRef         map[string]other.Tree
}

// mapsCollide holds two types of the same name from two packages that share a
// base name.
type mapsCollide struct {
	first  map[string]aliased.Nested
	second map[string]other.Nested
}

// collide references two types of the same name from two packages that
// share a base name, as a plain field, a pointer field and a slice element.
type collide struct {
	first       repa.Nested
	firstP      *repa.Nested
	firstSlice  []repa.Nested
	second      repb.Nested
	secondP     *repb.Nested
	secondSlice []repb.Nested
}

// collideEmbedFirst embeds the Nested type of package repa/report.
type collideEmbedFirst struct {
	repa.Nested
}

// collideEmbedSecond embeds the Nested type of package repb/report. That
// package has the same name as package repa/report.
type collideEmbedSecond struct {
	repb.Nested
}
