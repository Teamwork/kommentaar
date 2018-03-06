// Package openapi3 outputs to OpenAPI 3.
package openapi3

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/teamwork/kommentaar/docparse"
	yaml "gopkg.in/yaml.v2"
)

type (
	// OpenAPI output.
	OpenAPI struct {
		OpenAPI    string          `json:"openapi" yaml:"openapi"`
		Info       Info            `json:"info" yaml:"info"`
		Paths      map[string]Path `json:"paths" yaml:"paths"`
		Components Components      `json:"components" yaml:"components"`
	}

	// Info provides metadata about the API.
	Info struct {
		Title   string  `json:"title" yaml:"title"`
		Version string  `json:"version" yaml:"version"`
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
		Reference  string              `json:"$ref,omitempty" yaml:"$ref,omitempty"`
		Type       string              `json:"type,omitempty" yaml:"type,omitempty"`
		Properties map[string]Property `json:"properties,omitempty" yaml:"properties,omitempty"`
	}

	// Property ..
	Property struct {
		Description string `json:"description,omitempty" yaml:"description,omitempty"`
		Type        string `json:"type" yaml:"type"`
		Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
		//Format string `json:"format" yaml:"format"`
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

// Write to stdout.
func Write(w io.Writer, prog docparse.Program) error {
	out := OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   prog.Config.Title,
			Version: prog.Config.Version,
			Contact: Contact{
				Name:  prog.Config.ContactName,
				Email: prog.Config.ContactEmail,
				URL:   prog.Config.ContactSite,
			},
		},
		Paths:      map[string]Path{},
		Components: Components{Schemas: map[string]Schema{}},
	}

	// Add components.
	for k, v := range prog.References {
		// TODO
		// TODO: v.Info?
		_ = v
		out.Components.Schemas[k] = Schema{
			Properties: map[string]Property{},
		}

		for _, p := range v.Params {
			if k, ok := kindMap[p.Kind]; ok {
				p.Kind = k
			}

			// TODO
			if p.Kind != "string" && p.Kind != "boolean" && p.Kind != "integer" {
				continue
			}

			out.Components.Schemas[k].Properties[p.Name] = Property{
				Type:        p.Kind,
				Description: p.Info,
				//Required:    p.Required,
			}
		}
	}

	// Add endponts.
	var errList []string
	for _, e := range prog.Endpoints {
		op := Operation{
			Summary:     e.Tagline,
			Description: e.Info,
			OperationID: makeID(e),
			Tags:        e.Tags,
			Responses:   map[int]Response{},
		}

		addParams(&op.Parameters, "path", e.Request.Path)
		addParams(&op.Parameters, "query", e.Request.Query)
		addParams(&op.Parameters, "form", e.Request.Form)

		if e.Request.Body != nil {
			// TODO: store better.
			n := strings.Split(e.Request.Body.Reference, " ")
			op.RequestBody = RequestBody{
				Content: map[string]MediaType{
					e.Request.ContentType: MediaType{
						Schema: Schema{
							Reference: "#/components/schemas/" + n[len(n)-1],
						},
					},
				},
			}
		}

		// TODO: Should validate in docparse.
		// TODO: Should also validate there isn't the same status code twice.
		if len(e.Responses) == 0 {
			errList = append(errList, fmt.Sprintf(
				"%v: must have at least one response", e.Location))
			continue
		}
		for code, resp := range e.Responses {
			// TODO: add acual params.
			_ = resp
			op.Responses[code] = Response{
				Description: fmt.Sprintf("%v %v", code, http.StatusText(code)),
			}
		}

		switch e.Method {
		case "GET":
			out.Paths[e.Path] = Path{Get: op}
		case "POST":
			out.Paths[e.Path] = Path{Post: op}
		case "PUT":
			out.Paths[e.Path] = Path{Put: op}
		case "DELETE":
			out.Paths[e.Path] = Path{Delete: op}
		default:
			return fmt.Errorf("unknown method: %#v", e.Method)
		}
	}

	if len(errList) > 0 {
		return errors.New(strings.Join(errList, "\n"))
	}

	//d, err := json.MarshalIndent(&out, "", "  ")
	d, err := yaml.Marshal(&out)
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
	"":     "string",
	"int":  "integer",
	"bool": "boolean",
}

func addParams(list *[]Parameter, in string, params *docparse.Params) {
	if params == nil {
		return
	}

	// TODO: Support params.Reference

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

		*list = append(*list, Parameter{
			In:          in,
			Name:        p.Name,
			Required:    p.Required,
			Description: p.Info,
			Schema: Schema{
				Type: p.Kind,
			},
		})
	}
}

func makeID(e *docparse.Endpoint) string {
	return strings.Replace(fmt.Sprintf("%v_%v", e.Method,
		strings.Replace(e.Path, "/", "_", -1)), "__", "_", 1)
}
