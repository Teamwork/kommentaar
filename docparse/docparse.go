// Package docparse parses the comments.
package docparse // import "github.com/teamwork/kommentaar/docparse"

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/teamwork/utils/sliceutil"
	"github.com/teamwork/utils/stringutil"
	"golang.org/x/tools/go/loader"
)

// Endpoint denotes a single API endpoint.
type Endpoint struct {
	Method    string   // HTTP method (e.g. POST, DELETE, etc.)
	Path      string   // Request path.
	Tags      []string // Tags for grouping (optional).
	Tagline   string   // Single-line description (optional).
	Info      string   // More detailed description (optional).
	Request   Request
	Responses map[int]Response
}

// Request definition.
type Request struct {
	ContentType string // Content-Type that this request accepts for the body.
	Body        Params // Request body; usually a JSON body.
	Path        Params // Path parameters (e.g. /foo/:ID).
	Query       Params // Query parameters  (e.g. ?foo=id).
	Form        Params // Form parameters.
}

// Response definition.
type Response struct {
	ContentType string // Content-Type.
	Body        Params // Body.
}

// Params parameters for the path, query, form, request body, or response body.
// This can either be a list of parameters specified in the command, or a
// reference to a Go struct denoted with $object. You can't mix the two.
type Params struct {
	Params    []Param
	Reference Reference
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
	Name    string // Name of the struct (without package name).
	Package string // Package in which the struct resides.

	// TODO: Shouldn't store this in the Reference, but somewhere else (e.g. a
	// separate list of objects). This way it can be referenced more than once.
	Info   string  // Comment of the struct itself.
	Params []Param // Struct fields.
}

const headerDesc = "desc"

// TODO: allow some configuring of this.
var (
	defaultRequest  = "application/json"
	defaultResponse = "application/json"
)

var (
	reRequestHeader  = regexp.MustCompile(`Request body( \((.+?)\))?:`)
	reResponseHeader = regexp.MustCompile(`Response( (\d+?))?( \((.+?)\))?:`)
)

// Parse a single comment block.
func Parse(comment string) (*Endpoint, error) {
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
			e.Request.Path, err = parseParams(contents)
			if err != nil {
				return e, fmt.Errorf("could not parse path params: %v", err)
			}

		// Query:
		case header == "Query:":
			e.Request.Query, err = parseParams(contents)
			if err != nil {
				return e, fmt.Errorf("could not parse query params: %v", err)
			}

		// Form:
		case header == "Form:":
			e.Request.Form, err = parseParams(contents)
			if err != nil {
				return e, fmt.Errorf("could not parse form params: %v", err)
			}

		default:
			// Request body:
			// Request body (application/json):
			req := reRequestHeader.FindStringSubmatch(header)
			if req != nil {
				e.Request.ContentType = defaultRequest
				if len(req) == 3 && req[2] != "" {
					e.Request.ContentType = req[2]
				}

				e.Request.Body, err = parseParams(contents)
				if err != nil {
					return e, fmt.Errorf("could not parse request params: %v", err)
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

				r := Response{ContentType: defaultResponse}
				if len(resp) > 4 && resp[4] != "" {
					r.ContentType = resp[4]
				}

				r.Body, err = parseParams(contents)
				if err != nil {
					return e, fmt.Errorf("could not parse response %v params: %v", code, err)
				}

				if e.Responses == nil {
					e.Responses = make(map[int]Response)
				}
				e.Responses[int(code)] = r

				break
			}

			return e, fmt.Errorf("unknown header: %#v", header)
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
	paramOptional = "optional"
	paramRequired = "required"

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
//
// TODO: support $object
func parseParams(text string) (Params, error) {
	params := Params{}

	for _, line := range collapseIndents(text) {
		// Get ,-denoted tags from {..} block.
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
				return params, fmt.Errorf("invalid reference: %#v", line)
			}

			ref, err := getReference(strings.TrimSpace(s[1]))
			if err != nil {
				return params, err
			}

			params.Reference = ref

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

			case kindString, kindInt, kindBool, kindArrayString, kindArrayInt:
				p.Kind = t

			default:
				return params, fmt.Errorf("unknown parameter tag for %#v: %#v",
					p.Name, t)
			}
		}

		params.Params = append(params.Params, p)
	}

	if params.Reference.Name != "" && len(params.Params) > 0 {
		return params, errors.New("both a reference and parameters are given")
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

var progs map[string]*loader.Program

func init() { progs = make(map[string]*loader.Program) }

func loadProg(path string) (*loader.Program, error) {
	prog := progs[path]
	if prog != nil {
		return prog, nil
	}

	ctx := &build.Default
	conf := &loader.Config{
		Build:      ctx,
		ParserMode: parser.ParseComments,
	}
	//pkg, err := ctx.ImportDir(".", build.ImportComment)
	pkg, err := ctx.Import(path, ".", build.ImportComment)
	if err != nil {
		return nil, err
	}
	conf.ImportWithTests(pkg.ImportPath)
	prog, err = conf.Load()
	if err != nil {
		return nil, err
	}

	progs[path] = prog
	return prog, nil
}

// AnObject: "type AnObject struct" in the current package.
// github.com/foo/bar.AnObject: Same in another package.
//
//prog *loader.Program,
func getReference(path string) (Reference, error) {

	name := path
	pkg := "." // TODO: maybe use full pkg name?
	if c := strings.Index(name, " "); c > -1 {
		pkg = name[:c]
		name = name[c+1:]
	}

	prog, err := loadProg(pkg)
	if err != nil {
		return Reference{}, err
	}

	ref := Reference{Name: name, Package: pkg}

	for _, p := range prog.InitialPackages() {
		for _, f := range p.Files {
			for _, d := range f.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok {
					continue
				}

				found := false

				for _, s := range gd.Specs {
					ts, ok := s.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if ts.Name.Name != name {
						continue
					}

					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						return Reference{}, fmt.Errorf("%v is not a struct but a %T",
							name, ts.Type)
					}

					found = true

					if gd.Doc != nil {
						ref.Info = strings.TrimSpace(gd.Doc.Text())
					}

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

						p, err := parseParams(fmt.Sprintf("%v: %v", fName, doc))
						if err != nil {
							return Reference{}, fmt.Errorf("could not parse field %v for struct %v: %v",
								fName, name, err)
						}
						if len(p.Params) != 1 {
							return Reference{}, fmt.Errorf("len(p.Params) != 1 for field %v in struct %v: %#v",
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
								return Reference{}, fmt.Errorf("can't type assert")
							}

							p.Params[0].Kind = "[]" + elt.Name

						default:
							return Reference{}, fmt.Errorf("unknown type: %T", typ)
						}

						ref.Params = append(ref.Params, p.Params[0])
					} // range st.Fields.List
				} // range gd.Specs

				if found {
					goto end
				}
			} // range f.Decls
		} // range p.Files
	} // range prog.InitialPackages()

	return Reference{}, fmt.Errorf("could not find %v", path)

end:
	return ref, nil
}
