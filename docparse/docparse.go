// Package docparse parses the comments.
package docparse // import "github.com/teamwork/kommentaar/docparse"

import (
	"fmt"
	"go/ast"
	"go/token"
	"html/template"
	"io"
	"net/http"
	"os"
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
	Packages []string
	Output   func(io.Writer, *Program) error
	Debug    bool

	// General information.
	Title        string
	Description  template.HTML
	Version      string
	ContactName  string
	ContactEmail string
	ContactSite  string

	// Defaults.
	DefaultRequestCt  string
	DefaultResponseCt string
	DefaultResponse   map[int]Response
	Prefix            string
	Basepath          string
	StructTag         string
	MapTypes          map[string]string
	MapFormats        map[string]string
}

// DefaultResponse references.
type DefaultResponse struct {
	Lookup      string // e.g. models.Foo
	Description string // e.g. "200 OK"
	Schema      Schema
}

// NewProgram creates a new Program instance.
func NewProgram(dbg bool) *Program {
	printDebug = dbg

	// Clear cache; otherwise tests with -count 2 fail.
	// TODO: figure out why; should work really.
	declsCache = make(map[string][]declCache)

	return &Program{
		References: make(map[string]Reference),
		Config: Config{
			DefaultRequestCt:  "application/json",
			DefaultResponseCt: "application/json",
			MapTypes:          make(map[string]string),
			MapFormats:        make(map[string]string),

			// Override from commandline.
			Debug: dbg,
		},
	}
}

var printDebug bool

func dbg(s string, a ...interface{}) {
	if printDebug {
		_, _ = fmt.Fprintf(os.Stderr, "\x1b[38;5;244mdbg docparse: "+s+"\x1b[0m\n", a...)
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
	Path        *Ref   // Path parameters (e.g. /foo/{id}).
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
	IsEmbed bool    // Is an embedded struct.
	Schema  *Schema // JSON schema.

	Fields []Param // Struct fields.
}

const (
	refRef     = "$ref"
	refDefault = "$default"
	refEmpty   = "$empty"
	refData    = "$data"
)

var (
	reBasicHeader    = regexp.MustCompile(`^(Path|Form|Query): (.+)`)
	reRequestHeader  = regexp.MustCompile(`^Request body( \((.+?)\))?: (.+)`)
	reResponseHeader = regexp.MustCompile(`^Response( (\d+?))?( \((.+?)\))?: (.+)`)
)

// parseComment a single comment block in the file filePath.
func parseComment(prog *Program, comment, pkgPath, filePath string) ([]*Endpoint, int, error) {
	e := &Endpoint{}

	// Get start line and determine if this is a comment block.
	line1 := stringutil.GetLine(comment, 1)
	e.Method, e.Path, e.Tags = parseStartLine(line1)
	if e.Method == "" {
		return nil, 0, nil
	}

	// Find more start lines.
	i := 1
	start := len(line1)
	var aliases []*Endpoint
	for {
		l := stringutil.GetLine(comment, i+1)
		method, path, tags := parseStartLine(l)
		if method == "" {
			break
		}

		start += len(l)
		i++
		aliases = append(aliases, &Endpoint{
			Method: method,
			Path:   path,
			Tags:   tags,
		})
	}

	// Determine if the next line is the "tagline" (that is, a non-blank line).
	tagline := stringutil.GetLine(comment, i+1)
	if tagline != "" {
		e.Tagline = strings.TrimSpace(tagline)
		start += len(e.Tagline)
		i++
	}

	// Remove startlines and tagline from comment.
	comment = strings.TrimSpace(comment[start+i:])

	parsingDesc := false
	pastDesc := false
	var err error

	// Get description and Kommentaar directives.
	for _, line := range strings.Split(comment, "\n") {
		i++

		// Ignore empty lines unless we're parsing the description because it needs to include them
		if strings.TrimSpace(line) == "" && !parsingDesc {
			continue
		}

		// Form:
		// Query:
		// Path:
		h := reBasicHeader.FindStringSubmatch(line)
		if h != nil {
			parsingDesc = false
			pastDesc = true
			switch h[1] {
			case "Path":
				if e.Request.Path != nil {
					return nil, i, fmt.Errorf("%v already present", h[1])
				}
				e.Request.Path, err = parseRefLine(prog, "path", h[2], filePath)

				if err == nil {
					pathRef, err := GetReference(prog, "query", false, e.Request.Path.Reference, filePath)
					if err != nil {
						return nil, i, err
					}

					pp := PathParams(e.Path)
					for _, p := range pathRef.Fields {
						name := goutil.TagName(p.KindField, "path") // TODO: hardcoded path
						if name == "-" {
							continue
						}
						if name == "" {
							name = p.Name
						}

						if !sliceutil.InStringSlice(pp, name) {
							return nil, i, fmt.Errorf("parameter %q is not in the path %q",
								name, e.Path)
						}
					}
				}

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
		code, resp, err := ParseResponse(prog, filePath, line)
		if err != nil {
			return nil, i, err
		}
		if resp != nil {
			pastDesc = true
			if e.Responses == nil {
				e.Responses = make(map[int]Response)
			}

			if _, ok := e.Responses[code]; ok {
				return nil, i, fmt.Errorf("%v: response code %v defined more than once",
					e.Path, code)
			}

			e.Responses[code] = *resp
			continue
		}

		if pastDesc {
			return nil, i, fmt.Errorf("unknown directive: %#v", line)
		}

		parsingDesc = true
		e.Info += line + "\n"
	}

	e.Info = strings.TrimSpace(e.Info)

	if len(e.Responses) == 0 {
		return nil, 0, fmt.Errorf("%v: must have at least one response", e.Path)
	}

	r := make([]*Endpoint, len(aliases)+1)
	r[0] = e
	for i := range aliases {
		r[i+1] = aliases[i]
	}

	return r, 0, nil
}

var reParams = regexp.MustCompile(`{\w+}`)

// PathParams returns all {..} delimited path parameters.
func PathParams(path string) []string {
	if !strings.Contains(path, "{") {
		return nil
	}

	var params []string
	for _, p := range reParams.FindAllString(path, -1) {
		params = append(params, strings.Trim(p, "{}"))
	}

	return params
}

// ParseResponse parses a Response line.
//
// Exported so it can be used in the config, too.
func ParseResponse(prog *Program, filePath, line string) (int, *Response, error) {
	resp := reResponseHeader.FindStringSubmatch(line)
	if resp == nil {
		return 0, nil, nil
	}

	code := int64(http.StatusOK)
	if resp[1] != "" {
		var err error
		code, err = strconv.ParseInt(strings.TrimSpace(resp[1]), 10, 32)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid status code %#v: %v",
				resp[1], err)
		}
	}

	r := Response{ContentType: prog.Config.DefaultResponseCt}
	if resp[4] != "" {
		r.ContentType = resp[4]
	}

	var err error
	r.Body, err = parseRefLine(prog, "resp", resp[5], filePath)
	if err != nil {
		return 0, nil, fmt.Errorf("could not parse response %v params: %v", code, err)
	}

	codeText := fmt.Sprintf("%d %s", code, http.StatusText(int(code)))
	switch r.Body.Description {
	case "":
		r.Body.Description = codeText
	case refEmpty:
		r.Body.Description = codeText + " (no data)"
	case refData:
		if resp[4] == "" {
			return 0, nil, fmt.Errorf("explicit Content-Type required for $data in %v: %q",
				filePath, line)
		}

		r.Body.Description = fmt.Sprintf("%s (%s data)", codeText, r.ContentType)
	case refDefault:
		// Make sure it's defined.
		if _, ok := prog.Config.DefaultResponse[int(code)]; !ok {
			return 0, nil, fmt.Errorf("no default response for %v in %v: %q",
				code, filePath, line)
		}
		r.Body.Description = codeText
	}

	return int(code), &r, nil
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
	case refEmpty, refDefault, refData:
		params.Description = name // Filled in later.
	case refRef:
		s := strings.Split(line, ":")
		if len(s) != 2 {
			return nil, fmt.Errorf("invalid reference: %#v", line)
		}

		ref, err := GetReference(prog, context, false, strings.TrimSpace(s[1]), filePath)
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

// MapType maps some Go types to primitives, so they appear as such in the
// output. Most of the time users of the API don't really care if it's a
// "sql.NullString" or just a string.
func MapType(prog *Program, in string) (kind, format string) {
	if v, ok := prog.Config.MapTypes[in]; ok {
		kind = v
	}
	if v, ok := prog.Config.MapFormats[in]; ok {
		format = v
	}

	return kind, format
}
