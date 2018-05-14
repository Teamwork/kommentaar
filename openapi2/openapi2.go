// Package openapi2 outputs to OpenAPI 2.0
//
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md
// http://json-schema.org/
package openapi2

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/teamwork/kommentaar/docparse"
	yaml "gopkg.in/yaml.v2"
)

type (
	// OpenAPI output.
	OpenAPI struct {
		Swagger string `json:"swagger" yaml:"swagger"`
		Info    Info   `json:"info,omitempty" yaml:"info,omitempty"`

		// TODO: do we need this? wil lhave to come from config
		Host     string   `json:"host,omitempty" yaml:"host,omitempty"`
		BasePath string   `json:"basePath,omitempty" yaml:"basePath,omitempty"`
		Schemes  []string `json:"schemes,omitempty" yaml:"schemes,omitempty"`

		Paths map[string]*Path `json:"paths" yaml:"paths"`

		Parameters  map[string]Parameter       `json:"parameters" yaml:"parameters"`
		Definitions map[string]docparse.Schema `json:"definitions" yaml:"definitions"`

		Consumes []string `json:"consumes,omitempty" yaml:"consumes,omitempty"`
		Produces []string `json:"produces,omitempty" yaml:"produces,omitempty"`
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

	// Parameter describes a single operation parameter.
	Parameter struct {
		Name        string          `json:"name" yaml:"name"`
		In          string          `json:"in" yaml:"in"` // query, header, path, cookie
		Description string          `json:"description,omitempty" yaml:"description,omitempty"`
		Required    bool            `json:"required,omitempty" yaml:"required,omitempty"`
		Type        string          `json:"type,omitempty" yaml:"type,omitempty"`
		Format      string          `json:"format,omitempty" yaml:"format,omitempty"`
		Schema      docparse.Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
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
		Parameters  []interface{}    `json:"parameters,omitempty" yaml:"parameters,omitempty"`
		Responses   map[int]Response `json:"responses" yaml:"responses"`

		Consumes []string `json:"consumes,omitempty" yaml:"consumes,omitempty"`
		Produces []string `json:"produces,omitempty" yaml:"produces,omitempty"`
	}

	// Reference other components in the specification, internally and
	// externally.
	Reference struct {
		Ref string `json:"$ref" yaml:"$ref"`
	}

	// Response describes a single response from an API Operation.
	Response struct {
		Description string
		Schema      docparse.Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
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
		Swagger: "2.0",
		Info: Info{
			Title:   prog.Config.Title,
			Version: prog.Config.Version,
			Contact: Contact{
				Name:  prog.Config.ContactName,
				Email: prog.Config.ContactEmail,
				URL:   prog.Config.ContactSite,
			},
		},
		Paths:       map[string]*Path{},
		Parameters:  map[string]Parameter{},
		Definitions: map[string]docparse.Schema{},

		Consumes: []string{prog.Config.DefaultRequestCt},
		Produces: []string{prog.Config.DefaultRequestCt},
	}

	// Add components.
	for k, v := range prog.References {
		if v.Schema == nil {
			return fmt.Errorf("schema is nil for %v", k)
		}
		switch v.Context {
		case "form", "query", "path":
			// Nothing, this will be inline in the operation.
		default:
			out.Definitions[k] = *v.Schema
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

		if e.Request.Path != nil {
			ref := prog.References[e.Request.Path.Reference]
			for name, p := range ref.Schema.Properties {
				op.Parameters = append(op.Parameters, Parameter{
					Name:        name,
					In:          "path",
					Description: p.Description,
					Type:        p.Type,
					Required:    true,
				})
			}
		}
		if e.Request.Query != nil {
			ref := prog.References[e.Request.Query.Reference]
			for _, f := range ref.Fields {
				schema := ref.Schema.Properties[f.Name]
				op.Parameters = append(op.Parameters, Parameter{
					Name:        f.Name,
					In:          "query",
					Description: schema.Description,
					Type:        schema.Type,
					Required:    f.Required,
				})
			}
		}
		if e.Request.Form != nil {
			ref := prog.References[e.Request.Form.Reference]
			for _, f := range ref.Fields {
				schema := ref.Schema.Properties[f.Name]
				op.Parameters = append(op.Parameters, Parameter{
					Name:        f.Name,
					In:          "formData",
					Description: schema.Description,
					Type:        schema.Type,
					Required:    f.Required,
				})
			}
			op.Consumes = append(op.Consumes, "application/x-www-form-urlencoded")
		}

		// Add any {..} parameters in the path to the parameter list if they
		// haven't been specified manually in e.Request.Path.
		if strings.Contains(e.Path, "{") && e.Request.Path == nil {
			for _, param := range reParams.FindAllString(e.Path, -1) {
				param = strings.Trim(param, "{}")
				op.Parameters = append(op.Parameters, Parameter{
					Name:     param,
					In:       "path",
					Type:     "integer",
					Format:   "int64",
					Required: true,
				})
			}
		}

		if e.Request.Body != nil {
			op.Parameters = append(op.Parameters, Parameter{
				// TODO: name required, is there a better value to use?
				Name:        e.Request.Body.Reference,
				In:          "body",
				Description: e.Request.Body.Description,
				Required:    true,
				Schema: docparse.Schema{
					Reference: "#/definitions/" + e.Request.Body.Reference,
				},
			})
			op.Consumes = append(op.Consumes, e.Request.ContentType)
		}

		for code, resp := range e.Responses {
			r := Response{
				Description: fmt.Sprintf("%v %v", code, http.StatusText(code)),
			}

			// Link reference.
			if resp.Body != nil && resp.Body.Reference != "" {
				r.Schema = docparse.Schema{
					Reference: "#/definitions/" + resp.Body.Reference,
				}
			} else if dr, ok := prog.Config.DefaultResponse[code]; ok {
				lookup := strings.Split(dr.Lookup, "/")
				r.Schema = docparse.Schema{
					Reference: "#/definitions/" + lookup[len(lookup)-1],
				}
			}
			op.Responses[code] = r

			op.Produces = appendIfNotExists(op.Produces, resp.ContentType)
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

func makeID(e *docparse.Endpoint) string {
	return strings.Replace(fmt.Sprintf("%v_%v", e.Method,
		strings.Replace(e.Path, "/", "_", -1)), "__", "_", 1)
}

func appendIfNotExists(xs []string, y string) []string {
	for _, x := range xs {
		if x == y {
			return xs
		}
	}
	return append(xs, y)
}
