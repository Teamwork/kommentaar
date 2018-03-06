// Package docparse parses the comments.
package docparse // import "github.com/teamwork/kommentaar/docparse"

import (
	"errors"
	"fmt"
	"go/ast"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"

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
	DefaultRequest  string
	DefaultResponse string
	Title           string
	Version         string
	Contact         string
	Prefix          string
}

// Prog is the program we're currently working on.
var Prog Program

// InitProgram creates a new Program instance.
func InitProgram() {
	Prog = Program{
		References: make(map[string]Reference),
		Config: Config{
			DefaultRequest:  "application/json",
			DefaultResponse: "application/json",
		},
	}
}

// Endpoint denotes a single API endpoint.
type Endpoint struct {
	Method    string   // HTTP method (e.g. POST, DELETE, etc.)
	Path      string   // Request path.
	Tags      []string // Tags for grouping (optional).
	Tagline   string   // Single-line description (optional).
	Info      string   // More detailed description (optional).
	Location  string   // Location in the source we found this.
	Request   Request
	Responses map[int]Response
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
// reference to a Go struct denoted with $object. You can't mix the two.
type Params struct {
	Params []Param
	// Main reason to store as a string (and Refs as a map) for now is so that
	// it looks pretties in the pretty.Print() output. May not want to keep
	// this.
	Reference string //*Reference
}

// Param is a path, query, or form parameter.
type Param struct {
	Name     string // Parameter name
	Info     string // Detailed description
	Kind     string // Type information
	Required bool   // Is this required to always be sent?
}

// Reference to a Go struct.
type Reference struct {
	Name    string  // Name of the struct (without package name).
	Package string  // Package in which the struct resides.
	Info    string  // Comment of the struct itself.
	Params  []Param // Struct fields.
}

const headerDesc = "desc"

var (
	reRequestHeader  = regexp.MustCompile(`Request body( \((.+?)\))?:`)
	reResponseHeader = regexp.MustCompile(`Response( (\d+?))?( \((.+?)\))?:`)
)

// Parse a single comment block in the package pkg.
func Parse(comment, pkgName string) (*Endpoint, error) {
	e := &Endpoint{}

	line1 := stringutil.GetLine(comment, 1)
	line2 := stringutil.GetLine(comment, 2)
	if len(comment) >= len(line1)+len(line2)+1 {
		comment = strings.TrimSpace(comment[len(line1)+len(line2)+1:])
	} else {
		comment = strings.TrimSpace(comment[len(line1)+len(line2):])
	}

	e.Method, e.Path, e.Tags = getStartLine(line1)
	if e.Method == "" {
		return nil, nil
	}

	if Prog.Config.Prefix != "" {
		e.Path = path.Join(Prog.Config.Prefix, e.Path)
	}

	e.Tagline = strings.TrimSpace(line2)

	// Split the blocks.
	info, err := getBlocks(comment)
	if err != nil {
		return nil, err
	}

	for header, contents := range info {
		switch {

		// Initial description.
		case header == headerDesc:
			e.Info = contents

		// Path:
		case header == "Path:":
			e.Request.Path, err = parseParams(contents, pkgName)
			if err != nil {
				return nil, fmt.Errorf("could not parse path params: %v", err)
			}

		// Query:
		case header == "Query:":
			e.Request.Query, err = parseParams(contents, pkgName)
			if err != nil {
				return nil, fmt.Errorf("could not parse query params: %v", err)
			}

		// Form:
		case header == "Form:":
			e.Request.Form, err = parseParams(contents, pkgName)
			if err != nil {
				return nil, fmt.Errorf("could not parse form params: %v", err)
			}

		default:
			// Request body:
			// Request body (application/json):
			req := reRequestHeader.FindStringSubmatch(header)
			if req != nil {
				e.Request.ContentType = Prog.Config.DefaultRequest
				if len(req) == 3 && req[2] != "" {
					e.Request.ContentType = req[2]
				}

				e.Request.Body, err = parseParams(contents, pkgName)
				if err != nil {
					return nil, fmt.Errorf("could not parse request params: %v", err)
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

				r := Response{ContentType: Prog.Config.DefaultResponse}
				if len(resp) > 4 && resp[4] != "" {
					r.ContentType = resp[4]
				}

				r.Body, err = parseParams(contents, pkgName)
				if err != nil {
					return nil, fmt.Errorf("could not parse response %v params: %v", code, err)
				}

				if e.Responses == nil {
					e.Responses = make(map[int]Response)
				}
				e.Responses[int(code)] = r

				break
			}

			return nil, fmt.Errorf("unknown header: %#v", header)
		}
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

			header = line
			continue
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
func parseParams(text, pkgName string) (*Params, error) {
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

		// Reference another object.
		if name == "$object" {
			s := strings.Split(line, ":")
			if len(s) != 2 {
				return nil, fmt.Errorf("invalid reference: %#v", line)
			}

			_, path, err := getReference(strings.TrimSpace(s[1]), pkgName)
			if err != nil {
				return nil, err
			}

			// TODO: We store it as a path for now, as that's easier to debug in
			// the intermediate format (otherwise pretty.Print() show the full
			// object, which is kind of noisy). We should probably store it as a
			// pointer once I'm done with the docparse part.
			params.Reference = path

			continue
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

			case paramOmitEmpty:
				// TODO: implement this. Also load from struct tag?

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

// AnObject: "type AnObject struct" in the current package.
// github.com/foo/bar.AnObject: Same in another package.
func getReference(path, pkgName string) (*Reference, string, error) {
	name := path
	pkg := pkgName
	if c := strings.Index(name, " "); c > -1 {
		pkg = name[:c]
		name = name[c+1:]
	}
	path = fmt.Sprintf("%v %v", pkg, name)

	// Already parsed this one, don't need to do it again.
	if ref, ok := Prog.References[path]; ok {
		return &ref, "", nil
	}

	// Find type
	ts, err := FindType(pkg, name)
	if err != nil {
		return nil, "", err
	}

	// Make sure it's a struct.
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil, "", fmt.Errorf("%v is not a struct but a %T",
			name, ts.Type)
	}

	ref := Reference{Name: name, Package: pkg}
	if ts.Doc != nil {
		ref.Info = strings.TrimSpace(ts.Doc.Text())
	}

	// Parse all the fields.
	for _, f := range st.Fields.List {

		// TODO: why is names an array?
		fName := f.Names[0].Name

		// Doc is the comment above the field, Comment the
		// inline comment on the same line.
		var doc string
		if f.Doc != nil {
			doc = f.Doc.Text()
		} else if f.Comment != nil {
			doc = f.Comment.Text()
		}

		// Make sure that parseParams sees continued lines.
		// TODO: refactor a bit so we don't have to use this
		// hack.
		doc = strings.Replace(doc, "\n", "\n    ", -1)

		p, err := parseParams(fmt.Sprintf("%v: %v", fName, doc), pkgName)
		if err != nil {
			return nil, "", fmt.Errorf("could not parse field %v for struct %v: %v",
				fName, name, err)
		}
		if len(p.Params) != 1 {
			return nil, "", fmt.Errorf("len(p.Params) != 1 for field %v in struct %v: %#v",
				fName, name, p.Params)
		}

		switch typ := f.Type.(type) {
		case *ast.Ident:
			p.Params[0].Kind = typ.Name

		// TODO: this is kinda ugly. There's got to be a better
		// way?
		case *ast.ArrayType:
			elt, ok := typ.Elt.(*ast.Ident)
			if !ok {
				// e.g. "[]models.Language"
				slt, ok := typ.Elt.(*ast.SelectorExpr)
				if !ok {
					return nil, "", fmt.Errorf("can't type assert %T %[1]v", typ.Elt)
				}

				xid, ok := slt.X.(*ast.Ident)
				if !ok {
					return nil, "", fmt.Errorf("can't type assert selector %T %[1]v", slt.X)
				}

				p.Params[0].Kind = fmt.Sprintf("[]%v.%v", xid.Name, slt.Sel.Name)
			} else {
				// e.g. "[]Foo"
				p.Params[0].Kind = "[]" + elt.Name
			}

		// TODO: more uglyness.
		// e.g. "*models.Session"
		case *ast.StarExpr:
			xid, ok := typ.X.(*ast.SelectorExpr)
			if !ok {
				return nil, "", fmt.Errorf("can't type assert selector %T %[1]v", typ.X)
			}

			xid2, ok := xid.X.(*ast.Ident)
			if !ok {
				return nil, "", fmt.Errorf("can't type assert selector %T %[1]v", xid.X)
			}

			p.Params[0].Kind = fmt.Sprintf("*%v.%v", xid2.Name, xid.Sel.Name)

		default:
			return nil, "", fmt.Errorf("unknown type: %T", typ)
		}

		ref.Params = append(ref.Params, p.Params[0])
	}

	Prog.References[path] = ref
	return &ref, path, nil
}
