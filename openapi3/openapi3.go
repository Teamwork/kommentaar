// Package openapi3 outputs to OpenAPI 3.
package openapi3

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/teamwork/kommentaar/docparse"
)

type (
	// OpenAPI output.
	OpenAPI struct {
		OpenAPI string            `json:"openapi"`
		Info    OAInfo            `json:"info"`
		Paths   map[string]OAPath `json:"paths"`
	}

	// OAInfo ..
	OAInfo struct {
		Title   string `json:"title"`
		Version string `json:"version"`
	}

	// OAPath ..
	OAPath struct {
		Ref    string      `json:"ref,omitempty"`
		Get    OAOperation `json:"get,omitempty"`
		Post   OAOperation `json:"post,omitempty"`
		Put    OAOperation `json:"put,omitempty"`
		Delete OAOperation `json:"delete,omitempty"`
	}

	// OAOperation ..
	OAOperation struct {
		OperationID string        `json:"operationId"`
		Tags        []string      `json:"tags"`
		Summary     string        `json:"summary,omitempty"`
		Description string        `json:"description,omitempty"`
		Parameters  []OAParameter `json:"parameters,omitempty"`
		Body        OABody        `json:"requestBody,omitempty"`
		Responses   []OAResponse  `json:"responses"`
	}

	// OABody ..
	OABody struct {
		Content  map[string]OAMediaType `json:"content"`
		Required bool                   `json:"required,omitempty"`
	}

	// OAMediaType ..
	OAMediaType struct {
		Encoding   string `json:"encoding"`
		Properties map[string]OAProperty
		//Schema   ..
	}

	// OASchema ..
	//OASchema struct {
	//}

	// OAProperty ..
	OAProperty struct {
		Type string
	}

	// OAParameter ..
	OAParameter struct {
		Name        string `json:"name"`
		In          string `json:"in"` // query, header, path, cookie
		Description string `json:"description,omitempty"`
		Required    bool   `json:"required"`
	}

	// OAResponse ..
	OAResponse struct {
		Description string
		Content     map[string]OAMediaType `json:"content"`
	}
)

// Write to stdout.
func Write(w io.Writer, endpoints []*docparse.Endpoint) error {
	out := OpenAPI{
		OpenAPI: "3.0.0",
		Info: OAInfo{
			Title:   "Teamwork Desk", // TODO
			Version: "1.0",           // TODO
		},
		Paths: map[string]OAPath{},
	}

	for _, e := range endpoints {
		op := OAOperation{
			Summary:     e.Tagline,
			Description: e.Info,
			OperationID: makeID(e),
			Tags:        e.Tags,
		}

		// Path params
		for _, p := range e.Request.Path.Params {
			// TODO: Support refs
			op.Parameters = append(op.Parameters, OAParameter{
				In:          "path",
				Name:        p.Name,
				Required:    p.Required,
				Description: p.Info,
			})
		}

		// Query params
		for _, p := range e.Request.Query.Params {
			// TODO: Support refs
			op.Parameters = append(op.Parameters, OAParameter{
				In:          "query",
				Name:        p.Name,
				Required:    p.Required,
				Description: p.Info,
			})
		}

		// Form params
		for _, p := range e.Request.Form.Params {
			// TODO: Support refs
			op.Parameters = append(op.Parameters, OAParameter{
				In:          "form",
				Name:        p.Name,
				Required:    p.Required,
				Description: p.Info,
			})
		}

		// Request body
		fmt.Println(e.Request.Body) // TODO: write to w

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

	d, err := json.MarshalIndent(&out, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write(d)
	return err
}

func makeID(e *docparse.Endpoint) string {
	return strings.Replace(fmt.Sprintf("%v_%v", e.Method,
		strings.Replace(e.Path, "/", "_", -1)), "__", "_", 1)
}
