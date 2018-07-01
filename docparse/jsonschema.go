package docparse

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"path"
	"strconv"
	"strings"

	"github.com/teamwork/utils/goutil"
	"github.com/teamwork/utils/sliceutil"
)

type defaultType struct {
	value, kind string
}

func (t *defaultType) MarshalJSON() ([]byte, error) {
	switch t.kind {
	case "string":
		return json.Marshal(t.value)
	case "integer":
		n, err := strconv.ParseInt(t.value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("MarshalJSON: %v", err)
		}
		return json.Marshal(n)
	default:
		return nil, fmt.Errorf("MarshalJSON: unknown type: %v", t.kind)
	}
}

func (t *defaultType) MarshalYAML() (interface{}, error) {
	switch t.kind {
	case "string":
		return t.value, nil
	case "integer":
		n, err := strconv.ParseInt(t.value, 10, 64)
		if err != nil {
			err = fmt.Errorf("MarshalYAML: %v", err)
		}
		return n, err
	default:
		return "", fmt.Errorf("MarshalYAML: unknown type: %v", t.kind)
	}
}

func (t *defaultType) String() string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%v", t.value)
}

// The Schema Object allows the definition of input and output data types.
type Schema struct {
	Reference   string       `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Title       string       `json:"title,omitempty" yaml:"title,omitempty"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string       `json:"type,omitempty" yaml:"type,omitempty"`
	Enum        []string     `json:"enum,omitempty" yaml:"enum,omitempty"`
	Format      string       `json:"format,omitempty" yaml:"format,omitempty"`
	Required    []string     `json:"required,omitempty" yaml:"required,omitempty"`
	Default     *defaultType `json:"default,omitempty" yaml:"default,omitempty"`
	Minimum     int          `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum     int          `json:"maximum,omitempty" yaml:"maximum,omitempty"`

	// Store array items; for primitives:
	//   "items": {"type": "string"}
	// or custom types:
	//   "items": {"$ref": "#/definitions/positiveInteger"},
	Items *Schema `json:"items,omitempty" yaml:"items,omitempty"`

	// Store structs.
	Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
}

// Convert a struct to a JSON schema.
func structToSchema(prog *Program, name string, ref Reference) (*Schema, error) {
	schema := &Schema{
		Title:       name,
		Description: ref.Info,
		Type:        "object",
		Properties:  map[string]*Schema{},
	}

	for _, p := range ref.Fields {
		if p.KindField == nil {
			return nil, fmt.Errorf("p.KindField is nil for %v", name)
		}

		switch ref.Context {
		case "path", "query", "form":
			name = goutil.TagName(p.KindField, ref.Context)
		default:
			// TODO: doesn't have to be json tag; that's just what Desk happens to
			// use. We should get it from Content-Type or some such instead.
			name = goutil.TagName(p.KindField, "json")
		}

		if name == "-" {
			continue
		}
		if name == "" {
			name = p.Name
		}

		prop, err := fieldToSchema(prog, name, ref, p.KindField)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %v: %v", ref.Lookup, err)
		}

		// TODO: ugly
		if len(prop.Required) > 0 {
			switch ref.Context {
			case "path", "query", "form":
			// Do nothing
			default:
				name = goutil.TagName(p.KindField, ref.Context)
				schema.Required = append(schema.Required, name)
				prop.Required = nil
			}
		}

		if prop == nil {
			return nil, fmt.Errorf(
				"structToSchema: prop is nil for field %#v in %#v",
				name, ref.Lookup)
		}

		schema.Properties[name] = prop
	}

	return schema, nil
}

const (
	paramRequired  = "required"
	paramOptional  = "optional"
	paramOmitEmpty = "omitempty"
	paramReadOnly  = "readonly"
)

func setTags(name string, s *Schema, tags []string) error {
	for _, t := range tags {
		switch t {

		case paramRequired:
			s.Required = append(s.Required, name)
		case paramOptional:
			// Do nothing.
		case paramOmitEmpty:
			// TODO: implement this (also load from struct tag?), but I don't
			// see any way to do that in the OpenAPI spec?
			return fmt.Errorf("omitempty not implemented yet")
		// TODO
		case paramReadOnly:
			return fmt.Errorf("readonly not implemented yet")

		// Various string formats.
		// https://tools.ietf.org/html/draft-handrews-json-schema-validation-01#section-7.3
		case "date-time", "date", "time", "email", "idn-email", "hostname", "idn-hostname", "uri", "url":
			if t == "url" {
				t = "uri"
			}
			if t == "email" {
				t = "idn-email"
			}
			if t == "hostname" {
				t = "idn-hostname"
			}

			s.Format = t

		// Params with arguments.
		default:
			switch {
			case strings.HasPrefix(t, "enum: "):
				s.Type = "enum"
				for _, e := range strings.Split(t[5:], " ") {
					e = strings.TrimSpace(e)
					if e != "" {
						s.Enum = append(s.Enum, e)
					}
				}

			case strings.HasPrefix(t, "default: "):
				s.Default = &defaultType{strings.TrimSpace(t[8:]), s.Type}

			case strings.HasPrefix(t, "range: "):
				rng := strings.Split(t[6:], "-")
				if len(rng) != 2 {
					return fmt.Errorf("invalid range: %#v; must be as \"min-max\"", t)
				}
				rng[0] = strings.TrimSpace(rng[0])
				rng[1] = strings.TrimSpace(rng[1])

				if rng[0] != "" {
					n, err := strconv.ParseInt(rng[0], 10, 32)
					if err != nil {
						return fmt.Errorf("could not parse range minimum: %v", err)
					}
					s.Minimum = int(n)
				}
				if rng[1] != "" {
					n, err := strconv.ParseInt(rng[1], 10, 32)
					if err != nil {
						return fmt.Errorf("could not parse range maximum: %v", err)
					}
					s.Maximum = int(n)
				}

			default:
				return fmt.Errorf("unknown parameter tag for %#v: %#v",
					name, t)
			}
		}
	}

	return nil
}

// Convert a struct field to JSON schema.
func fieldToSchema(prog *Program, fName string, ref Reference, f *ast.Field) (*Schema, error) {
	var p Schema

	if f.Doc != nil {
		p.Description = f.Doc.Text()
	} else if f.Comment != nil {
		p.Description = f.Comment.Text()
	}
	p.Description = strings.TrimSpace(p.Description)

	var tags []string
	p.Description, tags = parseTags(p.Description)

	pkg := ref.Package
	var name *ast.Ident

	dbg("fieldToSchema: %v", f.Names)

	sw := f.Type
start:
	switch typ := sw.(type) {

	// Don't support interface{} for now. We'd have to add a lot of complexity
	// for it, and not sure if we're ever going to need it.
	case *ast.InterfaceType:
		return nil, errors.New("fieldToSchema: interface{} is not supported")

	// Pointer type; we don't really care about this for now, so just read over
	// it.
	case *ast.StarExpr:
		sw = typ.X
		goto start

	// Simple identifiers such as "string", "int", "MyType", etc.
	case *ast.Ident:
		canon, err := canonicalType(ref.File, pkg, typ)
		if err != nil {
			return nil, fmt.Errorf("cannot get canonical type: %v", err)
		}
		if canon != nil {
			sw = canon
			goto start
		}

		p.Type = JSONSchemaType(typ.Name)

		// e.g. string, int64, etc.: don't need to look up.
		if isPrimitive(p.Type) {
			goto end
		}

		p.Type = ""
		name = typ

	// An expression followed by a selector, e.g. "pkg.foo"
	case *ast.SelectorExpr:
		pkgSel, ok := typ.X.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("typ.X is not ast.Ident: %#v", typ.X)
		}

		pkg = pkgSel.Name
		name = typ.Sel

		canon, err := canonicalType(ref.File, pkgSel.Name, typ.Sel)
		if err != nil {
			return nil, fmt.Errorf("cannot get canonical type: %v", err)
		}
		if canon != nil {
			sw = canon
			goto start
		}

		lookup := pkg + "." + name.Name
		t, f := MapType(lookup)

		p.Format = f
		if t != "" {
			p.Type = JSONSchemaType(t)
			goto end
		}

		// Deal with array.
		// TODO: don't do this inline but at the end. Reason it doesn't work not
		// is because we always use GetReference().
		ts, _, _, err := findType(ref.File, pkg, name.Name)
		if err != nil {
			return nil, err
		}

		switch resolvType := ts.Type.(type) {
		case *ast.ArrayType:
			p.Type = "array"
			err := resolveArray(prog, ref, pkg, &p, resolvType.Elt)
			if err != nil {
				return nil, err
			}

			goto end
		}

	// Maps
	case *ast.MapType:
		// As far as I can find there is no obvious/elegant way to represent
		// this in JSON schema, so simply don't support it for now. I don't
		// think we actually use this anywhere.
		// TODO: We should really support this...
		//return nil, errors.New("fieldToSchema: maps are not supported due to JSON schema limitations")
		p.Type = "object"

	// Array and slices.
	case *ast.ArrayType:
		p.Type = "array"

		err := resolveArray(prog, ref, pkg, &p, typ.Elt)
		if err != nil {
			return nil, err
		}

		goto end

	default:
		return nil, fmt.Errorf("fieldToSchema: unknown type: %T", typ)
	}

	if name == nil {
		goto end
	}

end:
	// Check if the type resolves to a Go primitive.
	if name != nil {
		lookup := pkg + "." + name.Name
		t, err := getTypeInfo(prog, lookup, ref.File)
		if err != nil {
			return nil, err
		}
		if t != "" {
			p.Type = t
			if isPrimitive(p.Type) {
				goto end
			}
		}

		if i := strings.LastIndex(lookup, "/"); i > -1 {
			lookup = pkg[i+1:] + "." + name.Name
		}

		p.Description = "" // SwaggerHub will complain if both Description and $ref are set.
		p.Reference = lookup
	}

	err := setTags(fName, &p, tags)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func resolveArray(prog *Program, ref Reference, pkg string, p *Schema, typ ast.Expr) error {
	asw := typ

	var name *ast.Ident

arrayStart:
	switch typ := asw.(type) {

	// Ignore *
	case *ast.StarExpr:
		asw = typ.X
		goto arrayStart

	// Simple identifier: "string", "myCustomType".
	case *ast.Ident:

		dbg("resolveArray: ident: %#v", typ.Name)

		p.Items = &Schema{Type: JSONSchemaType(typ.Name)}

		if typ.Name == "byte" {
			p.Items = nil
			p.Type = "string"
			return nil
		}

		if isPrimitive(p.Items.Type) {
			return nil
		}

		p.Items.Type = ""
		name = typ

	// "pkg.foo"
	case *ast.SelectorExpr:

		dbg("resolveArray: selector: %#v -> %#v", typ.X, typ.Sel)

		pkgSel, ok := typ.X.(*ast.Ident)
		if !ok {
			return fmt.Errorf("typ.X is not ast.Ident: %#v", typ.X)
		}
		pkg = pkgSel.Name
		name = typ.Sel

	default:
		return fmt.Errorf("fieldToSchema: unknown array type: %T", typ)
	}

	// Check if the type resolves to a Go primitive.
	lookup := pkg + "." + name.Name
	t, err := getTypeInfo(prog, lookup, ref.File)
	if err != nil {
		return err
	}
	if t != "" {
		p.Type = t
		if isPrimitive(p.Type) {
			return nil
		}
	}

	if i := strings.LastIndex(pkg, "/"); i > -1 {
		lookup = pkg[i+1:] + "." + name.Name
	}
	p.Items = &Schema{Reference: lookup}

	// Add to prog.References.
	_, err = GetReference(prog, "", lookup, ref.File)
	return err
}

func isPrimitive(n string) bool {
	//"null", "boolean", "object", "array", "number", "string", "integer",
	return sliceutil.InStringSlice([]string{
		"null", "boolean", "number", "string", "integer",
	}, n)
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

// JSONSchemaType gets the type name as used in JSON schema.
func JSONSchemaType(t string) string {
	if m, ok := kindMap[t]; ok {
		return m
	}
	return t
}

func getTypeInfo(prog *Program, lookup, filePath string) (string, error) {
	var name, pkg string
	if c := strings.LastIndex(lookup, "."); c > -1 {
		// imported path: models.Foo
		pkg = lookup[:c]
		name = lookup[c+1:]
	} else {
		// Current package: Foo
		pkg = path.Dir(filePath)
		name = lookup
	}

	// Find type.
	ts, _, _, err := findType(filePath, pkg, name)
	if err != nil {
		return "", err
	}

	ident, ok := ts.Type.(*ast.Ident)
	if !ok {
		return "", nil
	}

	t := JSONSchemaType(ident.Name)
	return t, nil
}

// Get the canonical type.
func canonicalType(currentFile, pkgPath string, typ *ast.Ident) (ast.Expr, error) {
	if builtInType(typ.Name) {
		return nil, nil
	}

	var ts *ast.TypeSpec
	if typ.Obj == nil {
		var err error
		ts, _, _, err = findType(currentFile, pkgPath, typ.Name)
		if err != nil {
			return nil, err
		}
	} else {
		ts = typ.Obj.Decl.(*ast.TypeSpec)
	}

	// Don't resolve structs; we do this later.
	if _, ok := ts.Type.(*ast.StructType); ok {
		return nil, nil
	}

	return ts.Type, nil
}
