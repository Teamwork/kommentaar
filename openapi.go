package main

import (
	"fmt"
	"strings"

	"github.com/go-yaml/yaml"
)

type (
	// OpenAPI output.
	OpenAPI struct {
		OpenAPI string            `yaml:"openapi"`
		Info    OAInfo            `yaml:"info"`
		Paths   map[string]OAPath `yaml:"paths"`
	}

	// OAInfo ..
	OAInfo struct {
		Title   string `yaml:"title"`
		Version string `yaml:"version"`
	}

	// OAPath ..
	OAPath struct {
		Ref    string      `yaml:"ref,omitempty"`
		Get    OAOperation `yaml:"get,omitempty"`
		Post   OAOperation `yaml:"post,omitempty"`
		Put    OAOperation `yaml:"put,omitempty"`
		Delete OAOperation `yaml:"delete,omitempty"`
	}

	// OAOperation ..
	OAOperation struct {
		OperationID string        `yaml:"operationId"`
		Tags        []string      `yaml:"tags"`
		Summary     string        `yaml:"summary,omitempty"`
		Description string        `yaml:"description,omitempty"`
		Parameters  []OAParameter `yaml:"parameters,omitempty"`
		Body        OABody        `yaml:"requestBody,omitempty"`
		Responses   []OAResponse  `yaml:"responses"`
	}

	// OABody ..
	OABody struct {
		Content  map[string]OAMediaType `yaml:"content"`
		Required bool                   `yaml:"required,omitempty"`
	}

	// OAMediaType ..
	OAMediaType struct {
		Encoding   string `yaml:"encoding"`
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
		Name        string `yaml:"name"`
		In          string `yaml:"in"` // query, header, path, cookie
		Description string `yaml:"description,omitempty"`
		Required    bool   `yaml:"required"`
	}

	// OAResponse ..
	OAResponse struct {
		Description string
		Content     map[string]OAMediaType `yaml:"content"`
	}
)

func toOpenAPI3(endpoints []endpoint) error {
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
			Summary:     e.tagline,
			Description: e.info,
			OperationID: makeID(e),
			Tags:        e.tags,
		}

		for _, p := range e.request.query {
			// TODO: Support refs
			op.Parameters = append(op.Parameters, OAParameter{
				In:          "query",
				Name:        p.name,
				Required:    p.required,
				Description: p.info,
			})
		}

		switch e.method {
		case "GET":
			out.Paths[e.path] = OAPath{Get: op}
		case "POST":
			out.Paths[e.path] = OAPath{Post: op}
		case "PUT":
			out.Paths[e.path] = OAPath{Put: op}
		case "DELETE":
			out.Paths[e.path] = OAPath{Delete: op}
		default:
			return fmt.Errorf("unknown method: %#v", e.method)
		}
	}

	d, err := yaml.Marshal(&out)
	if err != nil {
		return err
	}

	fmt.Println(string(d))
	return nil
}

func makeID(e endpoint) string {
	return strings.Replace(fmt.Sprintf("%v_%v", e.method,
		strings.Replace(e.path, "/", "_", -1)), "__", "_", 1)
}
