// Package openapi3 outputs to OpenAPI 3.0.1
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
	"regexp"
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
		Schemas   map[string]docparse.Schema `json:"schemas" yaml:"schemas"`
		Responses map[int]Response           `json:"responses,omitempty" yaml:"responses,omitempty"`
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
		Schema docparse.Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	}

	// Parameter describes a single operation parameter.
	Parameter struct {
		Name        string          `json:"name" yaml:"name"`
		In          string          `json:"in" yaml:"in"` // query, header, path, cookie
		Description string          `json:"description,omitempty" yaml:"description,omitempty"`
		Required    bool            `json:"required,omitempty" yaml:"required,omitempty"`
		Schema      docparse.Schema `json:"schema" yaml:"schema"`
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

var reParams = regexp.MustCompile(`{\w+}`)

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
		Paths: map[string]*Path{},
		Components: Components{
			Schemas:   map[string]docparse.Schema{},
			Responses: map[int]Response{},
		},
	}

	// Add components.
	for k, v := range prog.References {
		if v.Schema == nil {
			return fmt.Errorf("schema is nil for %v", k)
		}

		out.Components.Schemas[k] = *v.Schema
	}

	// Add default responses.
	for k, v := range prog.Config.DefaultResponse {
		out.Components.Responses[k] = Response{
			Description: v.Schema.Description,
			Content:     map[string]MediaType{v.Description: {Schema: v.Schema}},
		}
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

		// Add any {..} parameters in the path to the parameter list.
		// OpenAPI spec mandates that they're defined as parameters, but 95% of
		// the time this is just pointless: "id is the id". Whoopdiedo, such
		// great docs.
		if strings.Contains(e.Path, "{") {
			for _, m := range reParams.FindAllString(e.Path, -1) {
				m = strings.Trim(m, "{}")

				found := false
				if e.Request.Path == nil {
					e.Request.Path = &docparse.Params{}
				}
				for _, p := range e.Request.Path.Params {
					if p.Name == m {
						found = true
						break
					}
				}

				if !found {
					e.Request.Path.Params = append(e.Request.Path.Params, docparse.Param{
						Name: m,
						Kind: "integer",
					})
				}
			}
		}

		addParams(&op.Parameters, "path", e.Request.Path)
		addParams(&op.Parameters, "query", e.Request.Query)
		addParams(&op.Parameters, "form", e.Request.Form)

		if e.Request.Body != nil {
			op.RequestBody = RequestBody{
				Content: map[string]MediaType{
					e.Request.ContentType: MediaType{
						Schema: docparse.Schema{
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

			// TODO: hack.
			if len(resp.Body.Params) > 0 && resp.Body.Params[0].Name == "$default" {
				r.Content = map[string]MediaType{
					resp.ContentType: MediaType{
						Schema: docparse.Schema{Reference: fmt.Sprintf("#/components/responses/%v", code)},
					},
				}
			}

			// Link reference.
			if resp.Body != nil && resp.Body.Reference != "" {
				r.Content = map[string]MediaType{
					resp.ContentType: MediaType{
						Schema: docparse.Schema{Reference: "#/components/schemas/" + resp.Body.Reference},
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

func addParams(list *[]Parameter, in string, params *docparse.Params) {
	if params == nil {
		return
	}

	// TODO: support references.
	//
	// This is annoying as fuck because Operation.Parameters is defined as
	// "[Parameter Object | Reference Object]", but this can't easily be
	// represented in Go :-/
	//
	// Perhaps it would be better to *always* use references? It would simplify
	// parsing quite a lot actually, and we don't use the inline syntax that
	// much.

	//if params.Reference != "" {
	//	list.Reference = p.Ref
	//	return
	//}

	for _, p := range params.Params {
		// Path parameters must have required set or SwaggerHub complains.
		if in == "path" {
			p.Required = true
			if p.Kind == "" {
				p.Kind = "integer"
			}
		}

		p.Kind = docparse.JSONSchemaType(p.Kind)

		s := docparse.Schema{Type: p.Kind, Format: p.Format}
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
