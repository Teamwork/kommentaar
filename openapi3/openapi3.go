// Package openapi3 outputs to OpenAPI 3.0
//
// https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md
// http://json-schema.org/
package openapi3 // import "github.com/teamwork/kommentaar/openapi3"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/imdario/mergo"
	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/utils/v2/goutil"
	"gopkg.in/yaml.v3"
)

const (
	specVersion  = "3.0.3"
	refPrefix    = "#/components/schemas/"
	oldRefPrefix = "#/definitions/"
	formCT       = "application/x-www-form-urlencoded"
)

type (
	// OpenAPI output.
	OpenAPI struct {
		OpenAPI string `json:"openapi" yaml:"openapi"`
		Info    Info   `json:"info" yaml:"info"`

		Servers    []Server         `json:"servers,omitempty" yaml:"servers,omitempty"`
		Tags       []Tag            `json:"tags,omitempty" yaml:"tags,omitempty"`
		Paths      map[string]*Path `json:"paths" yaml:"paths"`
		Components *Components      `json:"components,omitempty" yaml:"components,omitempty"`
	}

	// Info provides metadata about the API.
	Info struct {
		Title       string   `json:"title,omitempty" yaml:"title,omitempty"`
		Description string   `json:"description,omitempty" yaml:"description,omitempty"`
		Version     string   `json:"version,omitempty" yaml:"version,omitempty"`
		Contact     *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	}

	// Contact provides contact information for the exposed API.
	Contact struct {
		Name  string `json:"name,omitempty" yaml:"name,omitempty"`
		URL   string `json:"url,omitempty" yaml:"url,omitempty"`
		Email string `json:"email,omitempty" yaml:"email,omitempty"`
	}

	// Server the API is served from.
	Server struct {
		URL         string `json:"url" yaml:"url"`
		Description string `json:"description,omitempty" yaml:"description,omitempty"`
	}

	// Components holds reusable objects for the specification.
	Components struct {
		Schemas map[string]docparse.Schema `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	}

	// Parameter describes a single operation parameter.
	Parameter struct {
		Name        string           `json:"name" yaml:"name"`
		In          string           `json:"in" yaml:"in"`
		Description string           `json:"description,omitempty" yaml:"description,omitempty"`
		Required    bool             `json:"required,omitempty" yaml:"required,omitempty"`
		Schema      *docparse.Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	}

	// Tag adds metadata to a single tag that is used by the Operation type.
	Tag struct {
		Name string `json:"name" yaml:"name"`
	}

	// Path describes the operations available on a single path.
	Path struct {
		Get    *Operation `json:"get,omitempty" yaml:"get,omitempty"`
		Post   *Operation `json:"post,omitempty" yaml:"post,omitempty"`
		Put    *Operation `json:"put,omitempty" yaml:"put,omitempty"`
		Patch  *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
		Delete *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
		Head   *Operation `json:"head,omitempty" yaml:"head,omitempty"`
	}

	// Operation describes a single API operation on a path.
	Operation struct {
		OperationID string              `json:"operationId" yaml:"operationId"`
		Tags        []string            `json:"tags,omitempty" yaml:"tags,omitempty"`
		Summary     string              `json:"summary,omitempty" yaml:"summary,omitempty"`
		Description string              `json:"description,omitempty" yaml:"description,omitempty"`
		Parameters  []Parameter         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
		RequestBody *RequestBody        `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
		Responses   map[string]Response `json:"responses" yaml:"responses"`

		Extend map[string]interface{} `json:"-" yaml:"-"`
	}

	// RequestBody describes the body of a request.
	RequestBody struct {
		Description string               `json:"description,omitempty" yaml:"description,omitempty"`
		Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
		Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	}

	// MediaType describes a request or response body for one content type.
	MediaType struct {
		Schema *docparse.Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	}

	// Response describes a single response from an API Operation.
	Response struct {
		Description string               `json:"description" yaml:"description"`
		Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	}
)

var paramOrder = map[string]int{"path": 0, "query": 1, "header": 2, "cookie": 3}

func (o *Operation) toMap() (map[string]interface{}, error) {
	type Alias Operation
	data, err := json.Marshal((*Alias)(o))
	if err != nil {
		return nil, fmt.Errorf("json marshal: %v", err)
	}

	m := map[string]interface{}{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}

	if o.Extend != nil {
		if err := mergo.Merge(&m, o.Extend, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("merge extend: %v", err)
		}
	}
	return m, nil
}

// MarshalJSON implements the json.Marshaler interface.
func (o *Operation) MarshalJSON() ([]byte, error) {
	if o.Extend == nil {
		type Alias Operation
		return json.Marshal((*Alias)(o))
	}

	m, err := o.toMap()
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

// MarshalYAML implements the yaml.Marshaler interface.
func (o *Operation) MarshalYAML() (interface{}, error) {
	if o.Extend == nil {
		type Alias Operation
		return (*Alias)(o), nil
	}

	m, err := o.toMap()
	if err != nil {
		return nil, fmt.Errorf("toMap: %v", err)
	}
	return &m, nil
}

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
		OpenAPI: specVersion,
		Info: Info{
			Title:       prog.Config.Title,
			Description: string(prog.Config.Description),
			Version:     prog.Config.Version,
			Contact:     contact(prog),
		},
		Paths: map[string]*Path{},
	}
	if prog.Config.Basepath != "" {
		out.Servers = []Server{{URL: prog.Config.Basepath}}
	}

	seenTags := map[string]struct{}{}
	referenced := map[string]struct{}{}
	ref := func(s string) string {
		s = strings.TrimPrefix(s, oldRefPrefix)
		s = strings.TrimPrefix(s, refPrefix)
		referenced[s] = struct{}{}
		return refPrefix + s
	}

	for _, e := range prog.Endpoints {
		path := prog.Config.Prefix + e.Path

		op, err := buildOperation(prog, e, path, ref)
		if err != nil {
			return err
		}

		for _, t := range e.Tags {
			seenTags[t] = struct{}{}
		}

		if out.Paths[path] == nil {
			out.Paths[path] = &Path{}
		}
		if err := setOperation(out.Paths[path], e.Method, op); err != nil {
			return err
		}
	}

	if len(seenTags) > 0 {
		out.Tags = make([]Tag, 0, len(seenTags))
		for tag := range seenTags {
			out.Tags = append(out.Tags, Tag{Name: tag})
		}
		sort.Slice(out.Tags, func(i, j int) bool {
			return out.Tags[i].Name < out.Tags[j].Name
		})
	}

	schemas := map[string]docparse.Schema{}
	for k, v := range prog.References {
		if v.Schema == nil {
			return fmt.Errorf("schema is nil for %q", k)
		}
		prefixPropertyReferences(v.Schema.Properties, ref)
		schemas[k] = *v.Schema
	}
	for k := range schemas {
		if _, ok := referenced[k]; !ok {
			delete(schemas, k)
		}
	}
	if len(schemas) > 0 {
		out.Components = &Components{Schemas: schemas}
	}

	return marshal(outFormat, w, &out)
}

func contact(prog *docparse.Program) *Contact {
	if prog.Config.ContactName == "" && prog.Config.ContactEmail == "" && prog.Config.ContactSite == "" {
		return nil
	}
	return &Contact{
		Name:  prog.Config.ContactName,
		Email: prog.Config.ContactEmail,
		URL:   prog.Config.ContactSite,
	}
}

func buildOperation(prog *docparse.Program, e *docparse.Endpoint, path string,
	ref func(string) string) (*Operation, error) {

	op := &Operation{
		Summary:     e.Tagline,
		Description: e.Info,
		OperationID: makeID(e.Method, path),
		Tags:        e.Tags,
		Responses:   map[string]Response{},
		Extend:      e.Extend,
	}

	if e.Request.Path != nil {
		params, err := pathParams(prog, e.Request.Path, ref)
		if err != nil {
			return nil, err
		}
		op.Parameters = append(op.Parameters, params...)
	}

	if e.Request.Query != nil {
		params, err := structParams(prog, e.Request.Query, "query", ref)
		if err != nil {
			return nil, err
		}
		op.Parameters = append(op.Parameters, params...)
	}

	if strings.Contains(path, "{") && e.Request.Path == nil {
		for _, param := range docparse.PathParams(path) {
			op.Parameters = append(op.Parameters, Parameter{
				Name:     strings.Trim(param, "{}"),
				In:       "path",
				Required: true,
				Schema:   &docparse.Schema{Type: "integer"},
			})
		}
	}

	sort.SliceStable(op.Parameters, func(i, j int) bool {
		oi, oj := paramOrder[op.Parameters[i].In], paramOrder[op.Parameters[j].In]
		if oi != oj {
			return oi < oj
		}
		return op.Parameters[i].Name < op.Parameters[j].Name
	})

	body, err := requestBody(prog, e, ref)
	if err != nil {
		return nil, err
	}
	op.RequestBody = body

	setResponses(prog, e, op, ref)

	return op, nil
}

func pathParams(prog *docparse.Program, r *docparse.Ref, ref func(string) string) ([]Parameter, error) {
	def, ok := prog.References[r.Reference]
	if !ok || def.Schema == nil {
		return nil, fmt.Errorf("no schema for path params %q", r.Reference)
	}

	names := make([]string, 0, len(def.Schema.Properties))
	for name := range def.Schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)

	params := make([]Parameter, 0, len(names))
	for _, name := range names {
		p := def.Schema.Properties[name]
		description := p.Description
		if p.OmitDoc {
			description = ""
		}
		params = append(params, Parameter{
			Name:        name,
			In:          "path",
			Description: description,
			Required:    true,
			Schema:      paramSchema(p, ref),
		})
	}
	return params, nil
}

func structParams(prog *docparse.Program, r *docparse.Ref, in string,
	ref func(string) string) ([]Parameter, error) {

	def, ok := prog.References[r.Reference]
	if !ok || def.Schema == nil {
		return nil, fmt.Errorf("no schema for %s params %q", in, r.Reference)
	}

	var params []Parameter
	for _, f := range def.Fields {
		name := goutil.TagName(f.KindField, in)
		if name == "-" {
			continue
		}

		schema := def.Schema.Properties[name]
		if schema == nil {
			return nil, fmt.Errorf("schema is nil for %s field %q in %q", in, name, r.Reference)
		}
		if schema.OmitDoc {
			continue
		}

		params = append(params, Parameter{
			Name:        name,
			In:          in,
			Description: schema.Description,
			Required:    len(schema.Required) > 0,
			Schema:      paramSchema(schema, ref),
		})
	}
	return params, nil
}

func requestBody(prog *docparse.Program, e *docparse.Endpoint, ref func(string) string) (*RequestBody, error) {
	content := map[string]MediaType{}

	if e.Request.Body != nil {
		ct := e.Request.ContentType
		if ct == "" {
			ct = prog.Config.DefaultRequestCt
		}
		content[ct] = MediaType{Schema: &docparse.Schema{Reference: ref(e.Request.Body.Reference)}}
	}

	if e.Request.Form != nil {
		schema, err := formSchema(prog, e.Request.Form, ref)
		if err != nil {
			return nil, err
		}
		content[formCT] = MediaType{Schema: schema}
	}

	if len(content) == 0 {
		return nil, nil
	}

	body := &RequestBody{Required: true, Content: content}
	if e.Request.Body != nil {
		body.Description = e.Request.Body.Description
	}
	return body, nil
}

func formSchema(prog *docparse.Program, r *docparse.Ref, ref func(string) string) (*docparse.Schema, error) {
	def, ok := prog.References[r.Reference]
	if !ok || def.Schema == nil {
		return nil, fmt.Errorf("no schema for form params %q", r.Reference)
	}

	schema := &docparse.Schema{Type: "object", Properties: map[string]*docparse.Schema{}}
	for _, f := range def.Fields {
		name := goutil.TagName(f.KindField, "form")
		if name == "-" {
			continue
		}

		p := def.Schema.Properties[name]
		if p == nil {
			return nil, fmt.Errorf("schema is nil for form field %q in %q", name, r.Reference)
		}
		if p.OmitDoc {
			continue
		}

		field := paramSchema(p, ref)
		field.Description = p.Description
		schema.Properties[name] = field
		if len(p.Required) > 0 {
			schema.Required = append(schema.Required, name)
		}
	}
	sort.Strings(schema.Required)
	return schema, nil
}

func setResponses(prog *docparse.Program, e *docparse.Endpoint, op *Operation, ref func(string) string) {
	codes := make([]int, 0, len(e.Responses))
	for code := range e.Responses {
		codes = append(codes, code)
	}
	sort.Ints(codes)

	for _, code := range codes {
		resp := e.Responses[code]

		r := Response{}
		var schema *docparse.Schema
		if resp.Body != nil {
			r.Description = resp.Body.Description
			if resp.Body.Reference != "" {
				schema = &docparse.Schema{Reference: ref(resp.Body.Reference)}
			}
		}

		if schema == nil {
			if dr, ok := prog.Config.DefaultResponse[code]; ok && dr.Body != nil {
				schema = &docparse.Schema{Reference: ref(dr.Body.Reference)}
				if dr.ContentType != "" {
					resp.ContentType = dr.ContentType
				}
			}
		}

		if r.Description == "" {
			r.Description = statusText(code)
		}

		switch {
		case schema != nil:
			ct := resp.ContentType
			if ct == "" {
				ct = prog.Config.DefaultResponseCt
			}
			r.Content = map[string]MediaType{ct: {Schema: schema}}
		case resp.ContentType != "" && resp.ContentType != prog.Config.DefaultResponseCt:
			r.Content = map[string]MediaType{resp.ContentType: {}}
		}

		op.Responses[strconv.Itoa(code)] = r
	}
}

func paramSchema(s *docparse.Schema, ref func(string) string) *docparse.Schema {
	if s.Reference != "" {
		return &docparse.Schema{Reference: ref(s.Reference)}
	}

	out := &docparse.Schema{
		Type:     s.Type,
		Format:   s.Format,
		Enum:     s.Enum,
		Default:  s.Default,
		Minimum:  s.Minimum,
		Maximum:  s.Maximum,
		Readonly: s.Readonly,
	}

	if s.Items != nil {
		if s.Items.Reference != "" {
			out.Items = &docparse.Schema{Reference: ref(s.Items.Reference)}
		} else {
			out.Items = &docparse.Schema{
				Type:   s.Items.Type,
				Format: s.Items.Format,
				Enum:   s.Items.Enum,
			}
		}
	}

	if out.Type == "" {
		out.Type = "string"
	}
	return out
}

func setOperation(p *Path, method string, op *Operation) error {
	switch method {
	case http.MethodGet:
		p.Get = op
	case http.MethodPost:
		p.Post = op
	case http.MethodPut:
		p.Put = op
	case http.MethodPatch:
		p.Patch = op
	case http.MethodDelete:
		p.Delete = op
	case http.MethodHead:
		p.Head = op
	default:
		return fmt.Errorf("unknown method: %#v", method)
	}
	return nil
}

func statusText(code int) string {
	if t := http.StatusText(code); t != "" {
		return t
	}
	return "Response " + strconv.Itoa(code)
}

func marshal(outFormat string, w io.Writer, out *OpenAPI) error {
	var (
		d   []byte
		err error
	)
	switch outFormat {
	case "jsonindent":
		d, err = json.MarshalIndent(out, "", "  ")
	case "json":
		d, err = json.Marshal(out)
	case "yaml":
		var b bytes.Buffer
		yamlEncoder := yaml.NewEncoder(&b)
		yamlEncoder.SetIndent(2)
		err = yamlEncoder.Encode(out)
		d = b.Bytes()
	default:
		err = fmt.Errorf("unknown format: %#v", outFormat)
	}
	if err != nil {
		return err
	}

	if _, err := w.Write(d); err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}

func makeID(method, path string) string {
	return strings.Replace(fmt.Sprintf("%v_%v", method,
		strings.ReplaceAll(path, "/", "_")), "__", "_", 1)
}

func prefixPropertyReferences(properties map[string]*docparse.Schema, getRef func(string) string) {
	var rm []string
	for k, s := range properties {
		prefixSchemaReferences(s, getRef)

		if s.OmitDoc {
			rm = append(rm, k)
		}
	}

	for _, r := range rm {
		delete(properties, r)
	}
}

func prefixSchemaReferences(s *docparse.Schema, getRef func(string) string) {
	if s == nil {
		return
	}
	if s.Reference != "" {
		s.Reference = getRef(s.Reference)
	}
	prefixSchemaReferences(s.Items, getRef)
	prefixSchemaReferences(s.AdditionalProperties, getRef)
	if s.Properties != nil {
		prefixPropertyReferences(s.Properties, getRef)
	}
}
