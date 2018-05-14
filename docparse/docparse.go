// Package docparse parses the comments.
package docparse // import "github.com/teamwork/kommentaar/docparse"

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"net/http"
	"os"
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

	// JSON reference prefix for schemas, e.g #/components/schemas
	SchemaRefPrefix string

	// Defaults.
	DefaultRequestCt  string
	DefaultResponseCt string
	DefaultResponse   map[int]DefaultResponse
	Prefix            string
}

// DefaultResponse references.
type DefaultResponse struct {
	Lookup      string // e.g. models.Foo
	Description string // e.g. "200 OK"
	Schema      Schema
}

// NewProgram creates a new Program instance.
func NewProgram(dbg bool) *Program {
	debug = dbg

	return &Program{
		References: make(map[string]Reference),
		Config: Config{
			DefaultRequestCt:  "application/json",
			DefaultResponseCt: "application/json",
			SchemaRefPrefix:   "#/components/schemas",

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
	Pos, End  token.Position
}

// Request definition.
type Request struct {
	ContentType string // Content-Type that this request accepts for the body.
	Body        *Ref   // Request body; usually a JSON body.
	Path        *Ref   // Path parameters (e.g. /foo/:ID).
	Query       *Ref   // Query parameters  (e.g. ?foo=id).
	Form        *Ref   // Form parameters.
}

// Response definition.
type Response struct {
	ContentType string // Content-Type.
	Body        *Ref   // Body.
}

// Ref parameters for the path, query, form, request body, or response body.
// This can either be a list of parameters specified in the command, or a
// reference to a Go struct denoted with $ref. You can't mix the two.
type Ref struct {
	Description string
	// Main reason to store as a string (and Refs as a map) for now is so that
	// it looks pretties in the pretty.Print() output. May not want to keep
	// this.
	Reference string //*Reference
}

// Param is a path, query, or form parameter.
type Param struct {
	Name string // Parameter name
	//Info     string   // Detailed description
	//Required bool     // Is this required to always be sent?
	//Kind     string   // Type information
	//KindEnum []string // Enum fields, only when Kind=enum.
	//Format   string   // Format, such as "email", "date", etc.
	//Ref      string   // Reference something else; for Kind=struct and Kind=array.

	KindField *ast.Field // Type information from struct field.
}

// Reference to a Go struct.
type Reference struct {
	Name    string  // Name of the struct (without package name).
	Package string  // Package in which the struct resides.
	File    string  // File this struct resides in.
	Lookup  string  // Identifier as pkg.type.
	Info    string  // Comment of the struct itself.
	Context string  // Context we found it: path, query, form, req, resp.
	Schema  *Schema // JSON schema.

	Fields []Param // Struct fields.
}

var (
	reBasicHeader    = regexp.MustCompile(`^(Path|Form|Query): (.+)`)
	reRequestHeader  = regexp.MustCompile(`^Request body( \((.+?)\))?: (.+)`)
	reResponseHeader = regexp.MustCompile(`^Response( (\d+?))?( \((.+?)\))?: (.+)`)
)

// parseComment a single comment block in the file filePath.
func parseComment(prog *Program, comment, pkgPath, filePath string) (*Endpoint, int, error) {
	e := &Endpoint{}

	// Get start line and determine if this is a comment block.
	line1 := stringutil.GetLine(comment, 1)
	e.Method, e.Path, e.Tags = parseStartLine(line1)
	if e.Method == "" {
		return nil, 0, nil
	}

	// Determine if the second line is the "tagline".
	i := 1
	line2 := stringutil.GetLine(comment, 2)
	if line2 != "" {
		e.Tagline = strings.TrimSpace(line2)
		i++
	}
	comment = strings.TrimSpace(comment[len(line1)+len(line2)+1:])

	pastDesc := false
	var err error

	// Get description and Kommentaar directives.
	for _, line := range strings.Split(comment, "\n") {
		i++

		if strings.TrimSpace(line) == "" {
			continue
		}

		// Form:
		// Query:
		// Path:
		h := reBasicHeader.FindStringSubmatch(line)
		if h != nil {
			pastDesc = true
			switch h[1] {
			case "Path":
				if e.Request.Path != nil {
					return nil, i, fmt.Errorf("%v already present", h[1])
				}
				e.Request.Path, err = parseRefLine(prog, "path", h[2], filePath)
			case "Query":
				if e.Request.Query != nil {
					return nil, i, fmt.Errorf("%v already present", h[1])
				}
				e.Request.Query, err = parseRefLine(prog, "query", h[2], filePath)
			case "Form":
				if e.Request.Form != nil {
					return nil, i, fmt.Errorf("%v already present", h[1])
				}
				e.Request.Form, err = parseRefLine(prog, "form", h[2], filePath)
			}
			if err != nil {
				return nil, i, fmt.Errorf("could not parse %v params: %v", h[1], err)
			}

			continue
		}

		// Request body:
		// Request body (application/json):
		req := reRequestHeader.FindStringSubmatch(line)
		if req != nil {
			pastDesc = true
			if e.Request.Body != nil {
				return nil, i, fmt.Errorf("Request Body already present")
			}

			e.Request.ContentType = prog.Config.DefaultRequestCt
			if req[2] != "" {
				e.Request.ContentType = req[2]
			}

			e.Request.Body, err = parseRefLine(prog, "req", req[3], filePath)
			if err != nil {
				return nil, i, fmt.Errorf("could not parse request params: %v", err)
			}

			continue
		}

		// Response 200 (application/json):
		// Response 200:
		// Response:
		resp := reResponseHeader.FindStringSubmatch(line)
		if resp != nil {
			pastDesc = true
			code := int64(http.StatusOK)
			if resp[1] != "" {
				var err error
				code, err = strconv.ParseInt(strings.TrimSpace(resp[1]), 10, 32)
				if err != nil {
					return nil, i, fmt.Errorf("invalid status code %#v: %v",
						resp[1], err)
				}
			}

			r := Response{ContentType: prog.Config.DefaultResponseCt}
			if resp[4] != "" {
				r.ContentType = resp[4]
			}

			r.Body, err = parseRefLine(prog, "resp", resp[5], filePath)
			if err != nil {
				return nil, i, fmt.Errorf("could not parse response %v params: %v", code, err)
			}
			if r.Body.Description != "" {
				r.Body.Description = http.StatusText(int(code))
			}

			if e.Responses == nil {
				e.Responses = make(map[int]Response)
			}

			if _, ok := e.Responses[int(code)]; ok {
				return nil, i, fmt.Errorf("%v: response code %v defined more than once",
					e.Path, code)
			}

			e.Responses[int(code)] = r

			continue
		}

		if pastDesc {
			return nil, i, fmt.Errorf("unknown directive: %#v", line)
		}

		e.Info += line + "\n"
	}

	e.Info = strings.TrimSpace(e.Info)

	if len(e.Responses) == 0 {
		return nil, 0, fmt.Errorf("%v: must have at least one response", e.Path)
	}

	return e, 0, nil
}

var allMethods = []string{http.MethodGet, http.MethodHead, http.MethodPost,
	http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect,
	http.MethodOptions, http.MethodTrace}

// Get the first "start line" of a documentation block:
//   POST /path tag1 tag2
//
// The tags are optional, and the method is case-sensitive.
func parseStartLine(line string) (string, string, []string) {
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

// Process a Kommentaar directive line.
func parseRefLine(prog *Program, context, line, filePath string) (*Ref, error) {
	params := &Ref{}

	line = strings.TrimSpace(line)

	// Get tags from {..} blocks.
	line, tags := parseTags(line)
	_ = tags // TODO(param): Add this

	// Get description and name.
	var name, info string
	if colon := strings.Index(line, ":"); colon > -1 {
		name = line[:colon]
		info = strings.TrimSpace(line[colon+1:])
		_ = info // TODO(param): decide what to do with this
	} else {
		name = line
	}
	name = strings.TrimSpace(name)

	switch name {
	case "$empty":
		params.Description = "no data"
	case "$default":
		// Will be filled in later.
		// TODO: Move that code here!
		params.Description = "$default"
	case "$ref":
		s := strings.Split(line, ":")
		if len(s) != 2 {
			return nil, fmt.Errorf("invalid reference: %#v", line)
		}

		ref, err := GetReference(prog, context, strings.TrimSpace(s[1]), filePath)
		if err != nil {
			return nil, fmt.Errorf("GetReference: %v", err)
		}

		// We store it as a path for now, as that's easier to debug in the
		// intermediate format (otherwise pretty.Print() show the full
		// object, which is kind of noisy). We should probably store it as a
		// pointer once I'm done with the docparse part.
		params.Reference = ref.Lookup
	default:
		return nil, fmt.Errorf("invalid keyword: %#v", name)
	}

	return params, nil
}

// parseTags get tags from {..} blocks.
func parseTags(line string) (string, []string) {
	var alltags []string

	for {
		open := strings.Index(line, "{")
		if open == -1 {
			break
		}

		close := strings.Index(line, "}")
		if close == -1 {
			break
		}

		tags := strings.Split(line[open+1:close], ",")
		line = line[:open] + line[close+1:]

		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				alltags = append(alltags, tag)
			}
		}
	}

	// Remove multiple consecutive spaces.
	var pc rune
	var nl string
	for _, c := range line {
		if pc == ' ' && c == ' ' {
			pc = c
			continue
		}
		nl += string(c)
		pc = c
	}

	nl = strings.TrimRight(nl, "\n ")

	// So that:
	//   var: this is now documented {required}.
	// doesn't show with a space before the full stop:
	//   this is now documented .
	if strings.HasSuffix(nl, " .") {
		nl = nl[:len(nl)-2] + "."
	}

	return nl, alltags
}

var (
	mapTypes = map[string]string{
		// stdlib
		"sql.NullBool":    "bool",
		"sql.NullFloat64": "float64",
		"sql.NullInt64":   "int64",
		"sql.NullString":  "string",
		"time.Time":       "string",

		// http://github.com/guregu/null
		"null.Bool":   "bool",
		"null.Float":  "float64",
		"null.Int":    "int64",
		"null.String": "string",
		"null.Time":   "string",
		"zero.Bool":   "bool",
		"zero.Float":  "float64",
		"zero.Int":    "int64",
		"zero.String": "string",
		"zero.Time":   "string",

		// TODO: add this to config.
		"twnull.Bool":   "bool",
		"twnull.Int":    "int64",
		"twnull.String": "string",
		"twtime.Time":   "string",
	}

	mapFormats = map[string]string{
		"null.Time":   "date-time",
		"time.Time":   "date-time",
		"twtime.Time": "date-time",
		"zero.Time":   "date-time",
	}
)

// MapType maps some Go types to primitives, so they appear as such in the
// output. Most of the time users of the API don't really care if it's a
// "sql.NullString" or just a string.
func MapType(in string) (kind, format string) {
	if v, ok := mapTypes[in]; ok {
		kind = v
	}
	if v, ok := mapFormats[in]; ok {
		format = v
	}

	return kind, format
}

func builtInType(n string) bool {
	return sliceutil.InStringSlice([]string{"bool", "byte", "complex64",
		"complex128", "error", "float32", "float64", "int", "int8", "int16",
		"int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32",
		"uint64", "uintptr"}, n)
}
