// Package openapi3 outputs to OpenAPI 3.
//
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.1.md
// http://json-schema.org/
package openapi3

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/teamwork/kommentaar/docparse"
	yaml "gopkg.in/yaml.v2"
)

type (
	// OpenAPI output.
	OpenAPI struct {
		OpenAPI    string           `json:"openapi" yaml:"openapi"`
		Info       Info             `json:"info" yaml:"info"`
		Paths      map[string]*Path `json:"paths" yaml:"paths"`
		Components Components       `json:"components" yaml:"components"`
	}

	// Info provides metadata about the API.
	Info struct {
		Title   string  `json:"title,omitempty" yaml:"title,omitempty"`
		Version string  `json:"version,omitempty" yaml:"version,omitempty"`
		Contact Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	}

	// Contact provides contact information for the exposed API.
	Contact struct {
		Name  string `json:"name,omitempty" yaml:"name,omitempty"`
		URL   string `json:"url,omitempty" yaml:"url,omitempty"`
		Email string `json:"email,omitempty" yaml:"email,omitempty"`
	}

	// Components holds a set of reusable objects.
	Components struct {
		Schemas map[string]Schema `json:"schemas" yaml:"schemas"`
	}

	// Path describes the operations available on a single path.
	Path struct {
		Ref    string    `json:"ref,omitempty" yaml:"ref,omitempty"`
		Get    Operation `json:"get,omitempty" yaml:"get,omitempty"`
		Post   Operation `json:"post,omitempty" yaml:"post,omitempty"`
		Put    Operation `json:"put,omitempty" yaml:"put,omitempty"`
		Patch  Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
		Delete Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	}

	// Operation describes a single API operation on a path.
	Operation struct {
		OperationID string           `json:"operationId" yaml:"operationId"`
		Tags        []string         `json:"tags,omitempty" yaml:"tags,omitempty"`
		Summary     string           `json:"summary,omitempty" yaml:"summary,omitempty"`
		Description string           `json:"description,omitempty" yaml:"description,omitempty"`
		Parameters  []Parameter      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
		RequestBody RequestBody      `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
		Responses   map[int]Response `json:"responses" yaml:"responses"`
	}

	// RequestBody describes a single request body.
	RequestBody struct {
		Content  map[string]MediaType `json:"content" yaml:"content"`
		Required bool                 `json:"required,omitempty" yaml:"required,omitempty"`
	}

	// MediaType provides schema and examples for the media type identified by
	// its key.
	MediaType struct {
		Schema Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	}

	// The Schema Object allows the definition of input and output data types.
	Schema struct {
		Reference   string   `json:"$ref,omitempty" yaml:"$ref,omitempty"`
		Title       string   `json:"title,omitempty" yaml:"title,omitempty"`
		Description string   `json:"description,omitempty" yaml:"description,omitempty"`
		Type        string   `json:"type,omitempty" yaml:"type,omitempty"`
		Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
		Format      string   `json:"format,omitempty" yaml:"format,omitempty"`
		Required    []string `json:"required,omitempty" yaml:"required,omitempty"`

		// Store array items; for primitives:
		//   "items": {"type": "string"}
		// or custom types:
		//   "items": {"$ref": "#/definitions/positiveInteger"},
		Items *Schema `json:"items,omitempty" yaml:"items,omitempty"`

		// Store structs.
		Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	}

	// Parameter describes a single operation parameter.
	Parameter struct {
		Name        string `json:"name" yaml:"name"`
		In          string `json:"in" yaml:"in"` // query, header, path, cookie
		Description string `json:"description,omitempty" yaml:"description,omitempty"`
		Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
		Schema      Schema `json:"schema" yaml:"schema"`
	}

	// Response describes a single response from an API Operation.
	Response struct {
		Description string
		Content     map[string]MediaType `json:"content" yaml:"content"`
	}
)

// WriteYAML writes w as YAML.
func WriteYAML(w io.Writer, prog *docparse.Program) error {
	return write("yaml", w, prog)
}

// WriteJSON writes to w as JSON.
func WriteJSON(w io.Writer, prog *docparse.Program) error {
	return write("json", w, prog)
}

// WriteJSONIndent writes to w as indented JSON.
func WriteJSONIndent(w io.Writer, prog *docparse.Program) error {
	return write("jsonindent", w, prog)
}

func write(outFormat string, w io.Writer, prog *docparse.Program) error {

	out := OpenAPI{
		OpenAPI: "3.0.1",
		Info: Info{
			Title:   prog.Config.Title,
			Version: prog.Config.Version,
			Contact: Contact{
				Name:  prog.Config.ContactName,
				Email: prog.Config.ContactEmail,
				URL:   prog.Config.ContactSite,
			},
		},
		Paths:      map[string]*Path{},
		Components: Components{Schemas: map[string]Schema{}},
	}

	// Add components.
	for k, v := range prog.References {
		schema, err := structToSchema(prog, k, v)
		if err != nil {
			return err
		}
		out.Components.Schemas[k] = *schema
	}

	// Add endpoints.
	for _, e := range prog.Endpoints {
		e.Path = prog.Config.Prefix + e.Path

		op := Operation{
			Summary:     e.Tagline,
			Description: e.Info,
			OperationID: makeID(e),
			Tags:        e.Tags,
			Responses:   map[int]Response{},
		}

		// TODO: Support params.Reference for path, query, and form.
		addParams(&op.Parameters, "path", e.Request.Path)
		addParams(&op.Parameters, "query", e.Request.Query)
		addParams(&op.Parameters, "form", e.Request.Form)

		if e.Request.Body != nil {
			op.RequestBody = RequestBody{
				Content: map[string]MediaType{
					e.Request.ContentType: MediaType{
						Schema: Schema{
							Reference: "#/components/schemas/" + e.Request.Body.Reference,
						},
					},
				},
			}
		}

		for code, resp := range e.Responses {
			r := Response{
				Description: fmt.Sprintf("%v %v", code, http.StatusText(code)),
			}

			// Link reference.
			if resp.Body != nil && resp.Body.Reference != "" {
				r.Content = map[string]MediaType{
					resp.ContentType: MediaType{
						Schema: Schema{Reference: "#/components/schemas/" + resp.Body.Reference},
					},
				}
			}

			op.Responses[code] = r
		}

		if out.Paths[e.Path] == nil {
			out.Paths[e.Path] = &Path{}
		}

		switch e.Method {
		case "GET":
			out.Paths[e.Path].Get = op
		case "POST":
			out.Paths[e.Path].Post = op
		case "PUT":
			out.Paths[e.Path].Put = op
		case "PATCH":
			out.Paths[e.Path].Patch = op
		case "DELETE":
			out.Paths[e.Path].Delete = op
		default:
			return fmt.Errorf("unknown method: %#v", e.Method)
		}
	}

	var (
		d   []byte
		err error
	)
	switch outFormat {
	case "jsonindent":
		d, err = json.MarshalIndent(&out, "", "  ")
	case "json":
		d, err = json.Marshal(&out)
	case "yaml":
		d, err = yaml.Marshal(&out)
	default:
		err = fmt.Errorf("unknown format: %#v", outFormat)
	}
	if err != nil {
		return err
	}

	_, err = w.Write(d)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}

var kindMap = map[string]string{
	//"":     "string",
	"int":     "integer",
	"int8":    "integer",
	"int16":   "integer",
	"int32":   "integer",
	"int64":   "integer",
	"uint8":   "integer",
	"uint16":  "integer",
	"uint32":  "integer",
	"uint64":  "integer",
	"float32": "number",
	"float64": "number",
	"bool":    "boolean",
	"byte":    "string",
	"rune":    "string",
	"error":   "string",
}

func addParams(list *[]Parameter, in string, params *docparse.Params) {
	if params == nil {
		return
	}

	for _, p := range params.Params {
		// Path parameters must have required set or SwaggerHub complains.
		if in == "path" {
			p.Required = true
			if p.Kind == "" {
				p.Kind = "integer"
			}
		}

		if k, ok := kindMap[p.Kind]; ok {
			p.Kind = k
		}

		s := Schema{Type: p.Kind, Format: p.Format}
		if p.Kind == "enum" {
			s.Type = ""
			s.Enum = p.KindEnum
		}

		*list = append(*list, Parameter{
			In:          in,
			Name:        p.Name,
			Required:    p.Required,
			Description: p.Info,
			Schema:      s,
		})
	}
}

func makeID(e *docparse.Endpoint) string {
	return strings.Replace(fmt.Sprintf("%v_%v", e.Method,
		strings.Replace(e.Path, "/", "_", -1)), "__", "_", 1)
}

var debug = false

func dbg(s string, a ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, "\x1b[38;5;239mdbg openapi: "+s+"\x1b[0m\n", a...)
	}
}
