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
		OpenAPI string            `json:"openapi" yaml:"openapi"`
		Info    OAInfo            `json:"info" yaml:"info"`
		Paths   map[string]OAPath `json:"paths" yaml:"paths"`
	}

	// OAInfo ..
	OAInfo struct {
		Title   string    `json:"title" yaml:"title"`
		Version string    `json:"version" yaml:"version"`
		Contact OAContact `json:"contact,omitempty" yaml:"contact,omitempty"`
	}

	// OAContact ..
	OAContact struct {
		Name  string `json:"name,omitempty" yaml:"name,omitempty"`
		URL   string `json:"url,omitempty" yaml:"url,omitempty"`
		Email string `json:"email,omitempty" yaml:"email,omitempty"`
	}

	// OAPath ..
	OAPath struct {
		Ref    string      `json:"ref,omitempty" yaml:"ref,omitempty"`
		Get    OAOperation `json:"get,omitempty" yaml:"get,omitempty"`
		Post   OAOperation `json:"post,omitempty" yaml:"post,omitempty"`
		Put    OAOperation `json:"put,omitempty" yaml:"put,omitempty"`
		Delete OAOperation `json:"delete,omitempty" yaml:"delete,omitempty"`
	}

	// OAOperation describes a single API operation on a path.
	OAOperation struct {
		OperationID string             `json:"operationId" yaml:"operationId"`
		Tags        []string           `json:"tags,omitempty" yaml:"tags,omitempty"`
		Summary     string             `json:"summary,omitempty" yaml:"summary,omitempty"`
		Description string             `json:"description,omitempty" yaml:"description,omitempty"`
		Parameters  []OAParameter      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
		RequestBody OARequestBody      `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
		Responses   map[int]OAResponse `json:"responses" yaml:"responses"`
	}

	// OARequestBody describes a single request body.
	OARequestBody struct {
		Content  map[string]OAMediaType `json:"content" yaml:"content"`
		Required bool                   `json:"required,omitempty" yaml:"required,omitempty"`
	}

	// OAMediaType provides schema and examples for the media type identified by
	// its key.
	OAMediaType struct {
		Schema OASchema `json:"schema,omitempty" yaml:"schema,omitempty"`
		//Reference OAReference
	}

	// OAReference allows referencing other components in the specification,
	// internally and externally.
	OAReference struct {
		Ref string `json:"$ref" yaml:"$ref"`
	}

	// OASchema ..
	OASchema struct {
		Type      string `json:"type,omitempty" yaml:"type,omitempty"`
		Reference string `json:"$ref" yaml:"$ref"`
	}

	// OAParameter ..
	OAParameter struct {
		Name        string   `json:"name" yaml:"name"`
		In          string   `json:"in" yaml:"in"` // query, header, path, cookie
		Description string   `json:"description,omitempty" yaml:"description,omitempty"`
		Required    bool     `json:"required,omitempty" yaml:"required,omitempty"`
		Schema      OASchema `json:"schema" yaml:"schema"`
	}

	// OAResponse ..
	OAResponse struct {
		Description string
		Content     map[string]OAMediaType `json:"content" yaml:"content"`
	}
)

// Write to stdout.
func Write(w io.Writer, prog docparse.Program) error {
	out := OpenAPI{
		OpenAPI: "3.0.0",
		Info: OAInfo{
			Title:   prog.Config.Title,
			Version: prog.Config.Version,
			Contact: OAContact{
				Name:  prog.Config.ContactName,
				Email: prog.Config.ContactEmail,
				URL:   prog.Config.ContactSite,
			},
		},
		Paths: map[string]OAPath{},
	}

	var errList []string
	for _, e := range prog.Endpoints {
		op := OAOperation{
			Summary:     e.Tagline,
			Description: e.Info,
			OperationID: makeID(e),
			Tags:        e.Tags,
			Responses:   map[int]OAResponse{},
		}

		addParams(&op.Parameters, "path", e.Request.Path)
		addParams(&op.Parameters, "query", e.Request.Query)
		addParams(&op.Parameters, "form", e.Request.Form)

		if e.Request.Body != nil {
			op.RequestBody = OARequestBody{
				Content: map[string]OAMediaType{
					e.Request.ContentType: OAMediaType{
						Schema: OASchema{
							Reference: "#/components/schemas/Test", // TODO
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
			op.Responses[code] = OAResponse{
				Description: fmt.Sprintf("%v %v", code, http.StatusText(code)),
			}
		}

		switch e.Method {
		case "GET":
			out.Paths[e.Path] = OAPath{Get: op}
		case "POST":
			out.Paths[e.Path] = OAPath{Post: op}
		case "PUT":
			out.Paths[e.Path] = OAPath{Put: op}
		case "DELETE":
			out.Paths[e.Path] = OAPath{Delete: op}
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
	// TODO: clearly a placeholder
	_, _ = w.Write([]byte(`
components:
  schemas:
    Test:
      type: object
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
`))

	_, err = w.Write([]byte("\n"))
	return err
}

var kindMap = map[string]string{
	"":     "string",
	"int":  "integer",
	"bool": "boolean",
}

func addParams(list *[]OAParameter, in string, params *docparse.Params) {
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

		*list = append(*list, OAParameter{
			In:          in,
			Name:        p.Name,
			Required:    p.Required,
			Description: p.Info,
			Schema: OASchema{
				Type: p.Kind,
			},
		})
	}
}

func makeID(e *docparse.Endpoint) string {
	return strings.Replace(fmt.Sprintf("%v_%v", e.Method,
		strings.Replace(e.Path, "/", "_", -1)), "__", "_", 1)
}
