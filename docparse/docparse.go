// Package docparse parses the comments.
package docparse // import "github.com/teamwork/kommentaar/docparse"

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/teamwork/utils/goutil"
	"github.com/teamwork/utils/sliceutil"
	"github.com/teamwork/utils/stringutil"
)

// Program is the entire program: all collected endpoints and all collected
// references.
type Program struct {
	Config     Config
	Endpoints  []*Endpoint
	References map[string]Reference
}

// Config for the program.
type Config struct {
	// Kommentaar control.
	Paths  []string
	Output func(io.Writer, *Program) error
	Debug  bool

	// General information.
	Title        string
	Version      string
	ContactName  string
	ContactEmail string
	ContactSite  string

	// Defaults.
	DefaultRequest  string
	DefaultResponse string
	Prefix          string
}

// NewProgram creates a new Program instance.
func NewProgram(dbg bool) *Program {
	debug = dbg

	return &Program{
		References: make(map[string]Reference),
		Config: Config{
			DefaultRequest:  "application/json",
			DefaultResponse: "application/json",

			// Override from commandline.
			Debug: dbg,
		},
	}
}

var debug bool

func dbg(s string, a ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, "\x1b[38;5;244mdbg docparse: "+s+"\x1b[0m\n", a...)
	}
}

// Endpoint denotes a single API endpoint.
type Endpoint struct {
	Method    string   // HTTP method (e.g. POST, DELETE, etc.)
	Path      string   // Request path.
	Tags      []string // Tags for grouping (optional).
	Tagline   string   // Single-line description (optional).
	Info      string   // More detailed description (optional).
	Request   Request
	Responses map[int]Response

	//Location string // Location in the source we found the comment as "<file>:<line>:<col>"
	Pos, End token.Position
}

// Request definition.
type Request struct {
	ContentType string  // Content-Type that this request accepts for the body.
	Body        *Params // Request body; usually a JSON body.
	Path        *Params // Path parameters (e.g. /foo/:ID).
	Query       *Params // Query parameters  (e.g. ?foo=id).
	Form        *Params // Form parameters.
}

// Response definition.
type Response struct {
	ContentType string  // Content-Type.
	Body        *Params // Body.
}

// Params parameters for the path, query, form, request body, or response body.
// This can either be a list of parameters specified in the command, or a
// reference to a Go struct denoted with $ref. You can't mix the two.
type Params struct {
	Params      []Param
	Description string
	// Main reason to store as a string (and Refs as a map) for now is so that
	// it looks pretties in the pretty.Print() output. May not want to keep
	// this.
	Reference string //*Reference
}

// Param is a path, query, or form parameter.
type Param struct {
	Name      string     // Parameter name
	Info      string     // Detailed description
	Kind      string     // Type information
	KindField *ast.Field // Type information from struct field.
	Required  bool       // Is this required to always be sent?
}

// Reference to a Go struct.
type Reference struct {
	Name    string  // Name of the struct (without package name).
	Package string  // Package in which the struct resides.
	File    string  // File this struct resides in.
	Lookup  string  // Identifier as pkg.type.
	Info    string  // Comment of the struct itself.
	Params  []Param // Struct fields.
}

const headerDesc = "desc"

var (
	reRequestHeader  = regexp.MustCompile(`Request body( \((.+?)\))?:`)
	reResponseHeader = regexp.MustCompile(`Response( (\d+?))?( \((.+?)\))?:`)
)

// ParseComment a single comment block in the file filePath.
func ParseComment(prog *Program, comment, pkgPath, filePath string) (*Endpoint, error) {
	e := &Endpoint{}

	// Determine if this is a comment block.
	line1 := stringutil.GetLine(comment, 1)
	e.Method, e.Path, e.Tags = getStartLine(line1)
	if e.Method == "" {
		return nil, nil
	}

	// Determine if the second line is the "tagline"
	line2 := stringutil.GetLine(comment, 2)
	if len(comment) >= len(line1)+len(line2)+1 {
		comment = strings.TrimSpace(comment[len(line1)+len(line2)+1:])
	} else {
		comment = strings.TrimSpace(comment[len(line1)+len(line2):])
	}

	e.Tagline = strings.TrimSpace(line2)

	// Split the blocks.
	info, err := getBlocks(comment)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", e.Path, err)
	}

	for header, contents := range info {
		switch {

		// Initial description.
		case header == headerDesc:
			e.Info = contents

		// Path:
		case header == "Path:":
			e.Request.Path, err = parseParams(prog, contents, filePath)
			if err != nil {
				return nil, fmt.Errorf("could not parse path params: %v", err)
			}

		// Query:
		case header == "Query:":
			e.Request.Query, err = parseParams(prog, contents, filePath)
			if err != nil {
				return nil, fmt.Errorf("could not parse query params: %v", err)
			}

		// Form:
		case header == "Form:":
			e.Request.Form, err = parseParams(prog, contents, filePath)
			if err != nil {
				return nil, fmt.Errorf("could not parse form params: %v", err)
			}

		default:
			// Request body:
			// Request body (application/json):
			req := reRequestHeader.FindStringSubmatch(header)
			if req != nil {
				e.Request.ContentType = prog.Config.DefaultRequest
				if len(req) == 3 && req[2] != "" {
					e.Request.ContentType = req[2]
				}

				e.Request.Body, err = parseParams(prog, contents, filePath)
				if err != nil {
					return nil, fmt.Errorf("could not parse request params: %v", err)
				}
				if e.Request.Body.Reference == "" {
					return nil, fmt.Errorf("reference is mandatory for request body")
				}

				break
			}

			// Response 200 (application/json):
			// Response 200:
			// Response:
			resp := reResponseHeader.FindStringSubmatch(header)
			if resp != nil {
				code := int64(http.StatusOK)
				if len(resp) > 1 && resp[1] != "" {
					var err error
					code, err = strconv.ParseInt(strings.TrimSpace(resp[1]), 10, 32)
					if err != nil {
						return nil, fmt.Errorf("invalid status code %#v for %#v: %v",
							resp[1], header, err)
					}
				}

				r := Response{ContentType: prog.Config.DefaultResponse}
				if len(resp) > 4 && resp[4] != "" {
					r.ContentType = resp[4]
				}

				r.Body, err = parseParams(prog, contents, filePath)
				if err != nil {
					return nil, fmt.Errorf("could not parse response %v params: %v", code, err)
				}
				if r.Body.Reference == "" && r.Body.Description == "" {
					return nil, fmt.Errorf("reference is mandatory for response %v", code)
				}
				if r.Body.Description != "" {
					r.Body.Description = http.StatusText(int(code))
				}

				if e.Responses == nil {
					e.Responses = make(map[int]Response)
				}

				if _, ok := e.Responses[int(code)]; ok {
					return nil, fmt.Errorf("%v: response code %v defined more than once",
						e.Path, code)
				}

				e.Responses[int(code)] = r
				break
			}

			return nil, fmt.Errorf("unknown header: %#v", header)
		}
	}

	if len(e.Responses) == 0 {
		return nil, fmt.Errorf("%v: must have at least one response", e.Path)
	}

	return e, nil
}

var allMethods = []string{http.MethodGet, http.MethodHead, http.MethodPost,
	http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect,
	http.MethodOptions, http.MethodTrace}

// Get the first "start line" of a documentation block:
//   POST /path tag1 tag2
//
// The tags are optional, and the method is case-sensitive.
func getStartLine(line string) (string, string, []string) {
	words := strings.Fields(line)
	if len(words) < 2 || !sliceutil.InStringSlice(allMethods, words[0]) {
		return "", "", nil
	}

	// Make sure path starts with /.
	if len(words[1]) == 0 || words[1][0] != '/' {
		return "", "", nil
	}

	var tags []string
	if len(words) > 2 {
		tags = words[2:]
	}

	return words[0], words[1], tags
}

// Split the comment in to separate "blocks".
func getBlocks(comment string) (map[string]string, error) {
	info := map[string]string{}
	header := headerDesc

	for _, line := range strings.Split(comment, "\n") {
		// Blank lines.
		if len(line) == 0 {
			info[header] += "\n"
			continue
		}

		// New header.
		if line[0] != ' ' && strings.HasSuffix(line, ":") {
			var err error
			info, err = addBlock(info, header)
			if err != nil {
				return nil, err
			}

			if header == line {
				return nil, fmt.Errorf("duplicate header %#v", header)
			}

			header = line
			continue
		}

		// Single-line header, only supported with "$ref" and "$empty":
		//
		//  Response 200: $ref: AnObject
		//  Response 204: $empty
		//  Response 400: $ref: ErrorObject
		if line[0] != ' ' && (strings.Contains(line, ": $ref:") || strings.Contains(line, ": $empty")) {
			var err error
			info, err = addBlock(info, header)
			if err != nil {
				return nil, err
			}

			c := strings.Index(line, ":")
			header = line[:c+1]
			line = strings.TrimSpace(line[c+1:])
		}

		info[header] += line + "\n"
	}

	var err error
	info, err = addBlock(info, header)
	return info, err
}

func addBlock(info map[string]string, header string) (map[string]string, error) {
	if header == headerDesc {
		info[header] = strings.TrimSpace(info[header])
	} else {
		info[header] = strings.TrimRight(info[header], "\n")
	}

	if info[header] == "" || info[header] == "\n" {
		if header != headerDesc {
			return nil, fmt.Errorf("no content for header %#v", header)
		}
		delete(info, headerDesc)
	}

	return info, nil
}

const (
	paramOptional  = "optional"
	paramRequired  = "required"
	paramOmitEmpty = "omitempty"

	kindString      = "string"
	kindInt         = "int"
	kindBool        = "bool"
	kindArrayString = "[]string"
	kindArrayInt    = "[]int"
)

// Process one or more newline-separated parameters.
//
// A parameter looks like:
//
//   name
//   name: some description
//   name: {string, required}
//   name: some description {string, required}
func parseParams(prog *Program, text, filePath string) (*Params, error) {
	params := &Params{}

	for _, line := range collapseIndents(text) {
		// Get tags from {..} block.
		// TODO: What if there is more than one {..} block? I think we should
		// support this.
		var tags []string
		if open := strings.Index(line, "{"); open > -1 {
			if close := strings.Index(line, "}"); close > -1 {
				tags = strings.Split(line[open+1:close], ",")
				line = line[:open]
			}
		}
		// Allow empty {} block.
		if len(tags) == 1 && tags[0] == "" {
			tags = nil
		}

		// Get description and name.
		var name, info string
		if colon := strings.Index(line, ":"); colon > -1 {
			name = line[:colon]
			info = strings.TrimSpace(line[colon+1:])
		} else {
			name = line
		}
		name = strings.TrimSpace(name)

		// Reference a struct.
		if name == "$ref" {
			s := strings.Split(line, ":")
			if len(s) != 2 {
				return nil, fmt.Errorf("invalid reference: %#v", line)
			}

			ref, err := GetReference(prog, strings.TrimSpace(s[1]), filePath)
			if err != nil {
				return nil, fmt.Errorf("GetReference: %v", err)
			}

			// We store it as a path for now, as that's easier to debug in the
			// intermediate format (otherwise pretty.Print() show the full
			// object, which is kind of noisy). We should probably store it as a
			// pointer once I'm done with the docparse part.
			params.Reference = ref.Lookup

			continue
		} else if name == "$empty" {
			params.Description = "no data"
		}

		p := Param{Name: name, Info: info}

		// Validate tags.
		for _, t := range tags {
			switch strings.TrimSpace(t) {
			case paramRequired:
				p.Required = true
			// Bit redundant, but IMHO an explicit "optional" tag can clarify
			// things sometimes.
			case paramOptional:
				p.Required = false

			// TODO: implement this (also load from struct tag?), but I
			// don't see any way to do that in the OpenAPI spec?
			case paramOmitEmpty:
				return nil, fmt.Errorf("omitempty not implemented yet")

			// TODO: support {readonly} to indicate that it cannot be set by the
			// user.

			// TODO: suport {enum: foo, bar, xxx}

			// Only simple types are supported, and not tested types. Use a
			// struct if you wnat more advanced stuff (this feature is just
			// intended to quickly add a path parameter or query parameter or
			// two, without the bother of creating a "dead code" struct).
			// TODO: ideally I'd like to parse this a bit more flexible.
			// TODO: error out on multiple types being given
			case kindString, kindInt, kindBool, kindArrayString, kindArrayInt:
				p.Kind = t

			default:
				return nil, fmt.Errorf("unknown parameter tag for %#v: %#v",
					p.Name, t)
			}
		}

		params.Params = append(params.Params, p)
	}

	if params.Reference != "" && len(params.Params) > 0 {
		return nil, errors.New("both a reference and parameters are given")
	}

	return params, nil
}

func collapseIndents(in string) []string {
	var out []string
	prevIndent := 0

	for i, line := range strings.Split(in, "\n") {
		indent := getIndent(line)

		if i != 0 && indent > prevIndent {
			out[len(out)-1] += " " + strings.TrimSpace(line)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		prevIndent = indent
		out = append(out, line)
	}

	return out
}

func getIndent(s string) int {
	n := 0
	for _, c := range s {
		switch c {
		case ' ':
			n++
		case '\t':
			n += 8
		default:
			return n
		}
	}

	return n
}

// ErrNotStruct is used when GetReference resolves to something that is not a
// struct.
type ErrNotStruct struct {
	TypeSpec *ast.TypeSpec
	msg      string
}

func (err ErrNotStruct) Error() string {
	return err.msg
}

// GetReference finds a type by name. It can either be in the current path
// ("SomeStruct"), a package path with a type (e.g.
// "example.com/bar.SomeStruct"), or something from an imported package (e.g.
// "models.SomeStruct").
//
// References are stored in prog.References. This also finds and stores all
// nested references, so for:
//
//  type Foo struct {
//    Field Bar
//  }
//
// A GetReference("Foo", "") call will add two entries to prog.References: Foo
// and Bar (but only Foo is returned).
func GetReference(prog *Program, lookup, filePath string) (*Reference, error) {
	dbg("getReference: lookup: %#v -> filepath: %#v", lookup, filePath)

	var name, pkg string
	if c := strings.LastIndex(lookup, "."); c > -1 {
		// imported path: models.Foo
		pkg = lookup[:c]
		name = lookup[c+1:]
	} else {
		// Current package: Foo
		pkg = path.Dir(filePath)
		name = lookup
	}

	dbg("getReference: pkg: %#v -> name: %#v", pkg, name)

	// Already parsed this one, don't need to do it again.
	if ref, ok := prog.References[lookup]; ok {
		return &ref, nil
	}

	// Find type.
	ts, foundPath, pkg, err := FindType(filePath, pkg, name)
	if err != nil {
		return nil, err
	}

	// Make sure it's a struct.
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil, ErrNotStruct{ts, fmt.Sprintf(
			"%v is not a struct but a %T", name, ts.Type)}
	}

	ref := Reference{
		Name:    name,
		Package: pkg,
		Lookup:  filepath.Base(pkg) + "." + name,
		File:    foundPath,
	}
	if ts.Doc != nil {
		ref.Info = strings.TrimSpace(ts.Doc.Text())
	}

	// Parse all the fields.
	for _, f := range st.Fields.List {

		// Skip embedded struct for now.
		// TODO: we want to support this eventually.
		if len(f.Names) == 0 {
			continue
			//return nil, fmt.Errorf("embeded struct is not yet supported")
		}

		// Doc is the comment above the field, Comment the inline comment on the
		// same line.
		var doc string
		if f.Doc != nil {
			doc = f.Doc.Text()
		} else if f.Comment != nil {
			doc = f.Comment.Text()
		}

		// Make sure that parseParams sees continued lines.
		// TODO: split out "parseParams()" in two functions, so we don't have to
		// format it like this (same with Sprintf below).
		doc = strings.Replace(doc, "\n", "\n    ", -1)

		// Names is an array in cases like "Foo, Bar string".
		for _, fName := range f.Names {
			p, err := parseParams(prog, fmt.Sprintf("%v: %v", fName, doc), filePath)
			if err != nil {
				return nil, fmt.Errorf("could not parse field %v for struct %v: %v",
					fName, name, err)
			}

			// There should be only parameter per struct field. Something is
			// wrong if we found more (or fewer) than one.
			if len(p.Params) != 1 {
				return nil, fmt.Errorf("len(p.Params) != 1 for field %v in struct %v: %#v",
					fName, name, p.Params)
			}

			p.Params[0].KindField = f
			ref.Params = append(ref.Params, p.Params[0])
		}
	}

	prog.References[ref.Lookup] = ref

	// Scan all fields of f if it refers to a struct. Do this after storing the
	// reference in prog.References to prevent cyclic lookup issues.
	for _, f := range st.Fields.List {
		if goutil.TagName(f, "json") == "-" {
			continue
		}

		err := findNested(prog, f, foundPath, pkg)
		if err != nil {
			return nil, fmt.Errorf("\n  findNested: %v", err)
		}
	}

	return &ref, nil
}

// MapTypes maps some Go types to primitives, so they appear as such in the
// output. Most of the time users of the API don't really care if it's a
// "sql.NullString" or just a string.
var MapTypes = map[string]string{
	// stdlib
	"sql.NullBool":    "bool",
	"sql.NullFloat64": "float64",
	"sql.NullInt64":   "int64",
	"sql.NullString":  "string",
	"time.Time":       "string", // TODO: date

	// http://github.com/guregu/null
	"null.Bool":   "bool",
	"null.Float":  "float64",
	"null.Int":    "int64",
	"null.String": "String",
	"null.Time":   "string", // TODO: date
	"zero.Bool":   "bool",
	"zero.Float":  "float64",
	"zero.Int":    "int64",
	"zero.String": "String",
	"zero.Time":   "string", // TODO: date

	// TODO: add this to config.
	"twnull.Bool":   "bool",
	"twnull.Int":    "int64",
	"twnull.String": "string",
	"twtime.Time":   "string", // TODO: date
}

func findNested(prog *Program, f *ast.Field, filePath, pkg string) error {
	var name *ast.Ident

	sw := f.Type
start:
	switch typ := sw.(type) {

	// Pointer type; we don't really care about this for now, so just read over
	// it.
	case *ast.StarExpr:
		sw = typ.X
		goto start

	// Simple identifiers such as "string", "int", "MyType", etc.
	case *ast.Ident:
		if !builtInType(typ.Name) {
			name = typ
		}

	// An expression followed by a selector, e.g. "pkg.foo"
	case *ast.SelectorExpr:
		pkgSel, ok := typ.X.(*ast.Ident)
		if !ok {
			return fmt.Errorf("typ.X is not ast.Ident: %#v", typ.X)
		}
		name = typ.Sel
		pkg = pkgSel.Name

	// Array and slices.
	case *ast.ArrayType:
		asw := typ.Elt

	arrayStart:
		switch elementType := asw.(type) {

		// Ignore *
		case *ast.StarExpr:
			asw = elementType.X
			goto arrayStart

		// Simple identifier
		case *ast.Ident:
			if !builtInType(elementType.Name) {
				name = elementType
			}

		// "pkg.foo"
		case *ast.SelectorExpr:
			pkgSel, ok := elementType.X.(*ast.Ident)
			if !ok {
				return fmt.Errorf("elementType.X is not ast.Ident: %#v",
					elementType.X)
			}
			name = elementType.Sel
			pkg = pkgSel.Name
		}
	}

	if name == nil {
		return nil
	}

	lookup := pkg + "." + name.Name
	if i := strings.LastIndex(pkg, "/"); i > -1 {
		lookup = pkg[i+1:] + "." + name.Name
	}

	if _, ok := MapTypes[lookup]; ok {
		return nil
	}

	if _, ok := prog.References[lookup]; !ok {
		err := resolveType(prog, name, filePath, pkg)
		if err != nil {
			return fmt.Errorf("%v.%v: %v", pkg, name, err)
		}
	}
	return nil
}

func builtInType(n string) bool {
	return sliceutil.InStringSlice([]string{"bool", "byte", "complex64",
		"complex128", "error", "float32", "float64", "int", "int8", "int16",
		"int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32",
		"uint64", "uintptr"}, n)
}

func resolveType(prog *Program, typ *ast.Ident, filePath, pkg string) error {

	var ts *ast.TypeSpec
	if typ.Obj == nil {
		var err error
		ts, _, _, err = FindType(filePath, pkg, typ.Name)
		if err != nil {
			return err
		}
	} else {
		var ok bool
		ts, ok = typ.Obj.Decl.(*ast.TypeSpec)
		if !ok {
			return fmt.Errorf("resolveType: not a type declaration but %T",
				typ.Obj.Decl)
		}
	}

	// Only need to add struct types. Stuff like "type Foo string" gets added as
	// simply a string.
	_, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil
	}

	// This sets prog.References
	lookup := pkg + "." + typ.Name
	_, err := GetReference(prog, lookup, filePath)
	return err
}
