// Package docparse parses the comments.
package docparse

import (
	"fmt"
	"go/ast"
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

// Request ..
type Request struct {
	ContentType string
	Path        []Param
	Query       []Param
	Form        []Param
	Body        Reference
}

// Response ..
type Response struct {
	ContentType string
	Body        Reference
}

// Reference ..
type Reference struct {
	obj string
}

// Object ..
type Object struct {
}

// Param ..
type Param struct {
	Name     string // Parameter name
	Info     string // Detailed description
	Kind     string // Type information
	Required bool   // Is this required to always be sent?
	//ref      string
}

// TODO: allow some configuring of this.
var (
	defaultRequest  = "application/json"
	defaultResponse = "application/json"
)

var (
	reRequestHeader  = regexp.MustCompile(`Request body( \((.+?)\))?:`)
	reResponseHeader = regexp.MustCompile(`Response body( (\d+?))?( \((.+?)\))?:`)
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
		case header == "desc":
			e.Info = contents

		// Path:
		case header == "Path:":
			e.Request.Path, err = parseParams(contents)
			if err != nil {
				return e, err
			}

		// Query:
		case header == "Query:":
			e.Request.Query, err = parseParams(contents)
			if err != nil {
				return e, err
			}

		// Form:
		case header == "Form:":
			e.Request.Form, err = parseParams(contents)
			if err != nil {
				return e, err
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

				//e.Request.body, err = getReference(prog, contents)
				//if err != nil {
				//	return e, err
				//}
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
				if e.Responses == nil {
					e.Responses = make(map[int]Response)
				}
				e.Responses[int(code)] = r

				//e.response.body, err = getReference(prog, info["resBody"])
				//if err != nil {
				//	return e, nil
				//}
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
	header := "desc"

	for _, line := range strings.Split(comment, "\n") {
		// Blank lines.
		if len(line) == 0 {
			info[header] += "\n"
			continue
		}

		// New header.
		if line[0] != ' ' && strings.HasSuffix(line, ":") {
			if header == "desc" {
				info[header] = strings.TrimSpace(info[header])
			} else {
				info[header] = strings.TrimRight(info[header], "\n")
			}
			if info[header] == "" || info[header] == "\n" {
				if header != "desc" {
					return nil, fmt.Errorf("no content for header %#v", header)
				}
				delete(info, "desc")
			}
			header = line
			continue
		}

		info[header] += line + "\n"
	}

	if header == "desc" {
		info[header] = strings.TrimSpace(info[header])
	} else {
		info[header] = strings.TrimRight(info[header], "\n")
	}
	if info[header] == "" || info[header] == "\n" {
		if header != "desc" {
			return nil, fmt.Errorf("no content for header %#v", header)
		}
		delete(info, "desc")
	}

	return info, nil
}

// Process one or more newline-separated parameters.
//
// A parameter looks like:
//
//   name
//   name: some description
//   name: (string, required)
//   name: some description (string, required)
func parseParams(text string) ([]Param, error) {
	params := []Param{}

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p := Param{}

		// Get tags
		var tags []string
		if open := strings.Index(line, "("); open > -1 {
			if strings.HasSuffix(line, ")") {
				tags = strings.Split(line[open+1:len(line)-1], ",")
				line = line[:open]
			}
		}

		// Get description and name
		if colon := strings.Index(line, ":"); colon > -1 {
			p.Name = line[:colon]
			p.Info = strings.TrimSpace(line[colon+1:])
		} else {
			p.Name = line
		}
		p.Name = strings.TrimSpace(p.Name)

		for _, t := range tags {
			t = strings.TrimSpace(t)
			switch {
			case t == "required":
				p.Required = true
			//case strings.HasPrefix(t, "object:"):
			//	p.ref = strings.TrimSpace(strings.Split(t, ":")[1])
			default:
				p.Kind = t
			}
		}

		params = append(params, p)
	}

	return params, nil
}

func getReference(prog *loader.Program, text string) (Reference, error) {
	text = strings.TrimSpace(text)
	ref := Reference{}

	if !strings.HasPrefix(text, "object:") {
		return ref, fmt.Errorf("must be an object reference: %v", text)
	}
	ref.obj = strings.TrimSpace(strings.Split(text, ":")[1])

	for _, p := range prog.InitialPackages() {
		for _, f := range p.Files {
			for _, d := range f.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok {
					continue
				}

				if gd.Doc != nil {
					fmt.Printf("%s\n", gd.Doc.Text())

					for _, s := range gd.Specs {
						for _, f := range s.(*ast.TypeSpec).Type.(*ast.StructType).Fields.List {
							fmt.Printf("%#v - %#v\n", f.Names[0].Name, f.Doc.Text())
						}
					}
				}
			}
		}

	}

	return ref, nil
}
