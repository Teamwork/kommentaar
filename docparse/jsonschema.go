package docparse

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/teamwork/utils/goutil"
	"github.com/teamwork/utils/sliceutil"
	yaml "gopkg.in/yaml.v2"
)

// The Schema Object allows the definition of input and output data types.
type Schema struct {
	Reference   string   `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Title       string   `json:"title,omitempty" yaml:"title,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string   `json:"type,omitempty" yaml:"type,omitempty"`
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Format      string   `json:"format,omitempty" yaml:"format,omitempty"`
	Required    []string `json:"required,omitempty" yaml:"required,omitempty"`
	Default     string   `json:"default,omitempty" yaml:"default,omitempty"`
	Minimum     int      `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum     int      `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	Readonly    *bool    `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`

	// Store array items; for primitives:
	//   "items": {"type": "string"}
	// or custom types:
	//   "items": {"$ref": "#/definitions/positiveInteger"},
	Items *Schema `json:"items,omitempty" yaml:"items,omitempty"`

	// Store structs.
	Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	// We will not forbid to add propreties to an struct, so instead of using the
	// bool value, we use the schema definition
	AdditionalProperties *Schema `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`

	OmitDoc      bool   `json:"-" yaml:"-"` // {omitdoc}
	CustomSchema string `json:"-" yaml:"-"` // {schema: path}
}

// Convert a struct to a JSON schema.
func structToSchema(prog *Program, name, tagName string, ref Reference) (*Schema, error) {
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

		name = goutil.TagName(p.KindField, tagName)

		if name == "-" {
			continue
		}
		if name == "" {
			name = p.Name
		}

		prop, err := fieldToSchema(prog, name, tagName, ref, p.KindField)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %v: %v", ref.Lookup, err)
		}

		if !sliceutil.InStringSlice([]string{"path", "query", "form"}, ref.Context) {
			fixRequired(schema, prop)
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

// The required tags are added to the property itself, rather than to the
// parent. So fix that by moving it from "prop" to "parent".
//
// TODO: fix it so we don't have to do this.
func fixRequired(parent *Schema, prop *Schema) {
	parent.Required = append(parent.Required, prop.Required...)
	prop.Required = nil

	for _, p := range prop.Properties {
		fixRequired(prop, p)
	}
}

const (
	paramRequired  = "required"
	paramOptional  = "optional"
	paramOmitEmpty = "omitempty"
	paramReadOnly  = "readonly"
	paramOmitDoc   = "omitdoc"
)

func setTags(name, fName string, p *Schema, tags []string) error {
	for _, t := range tags {
		switch t {

		case paramOmitDoc:
			p.OmitDoc = true
		case paramRequired:
			p.Required = append(p.Required, name)
		case paramOptional:
			// Do nothing.
		case paramOmitEmpty:
			// TODO: implement this (also load from struct tag?), but I don't
			// see any way to do that in the OpenAPI spec?
			return fmt.Errorf("omitempty not implemented yet")
		case paramReadOnly:
			t := true
			p.Readonly = &t

		// Various string formats.
		// https://tools.ietf.org/html/draft-handrews-json-schema-validation-01#section-7.3
		case "datetime", "date-time", "date", "time", "email", "idn-email", "hostname", "idn-hostname", "uri", "url":
			if t == "datetime" {
				t = "date-time"
			}
			if t == "url" {
				t = "uri"
			}
			if t == "email" {
				t = "idn-email"
			}
			if t == "hostname" {
				t = "idn-hostname"
			}

			p.Format = t

		// Params with arguments.
		default:
			switch {
			case strings.HasPrefix(t, "enum: "):
				p.Type = "enum"
				for _, e := range strings.Split(t[5:], " ") {
					e = strings.TrimSpace(e)
					if e != "" {
						p.Enum = append(p.Enum, e)
					}
				}

			case strings.HasPrefix(t, "default: "):
				p.Default = strings.TrimSpace(t[8:])

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
					p.Minimum = int(n)
				}
				if rng[1] != "" {
					n, err := strconv.ParseInt(rng[1], 10, 32)
					if err != nil {
						return fmt.Errorf("could not parse range maximum: %v", err)
					}
					p.Maximum = int(n)
				}

			case strings.HasPrefix(t, "schema: "):
				p.CustomSchema = filepath.Join(filepath.Dir(fName), t[8:])
				err := readAndUnmarshalSchemaFile(p.CustomSchema, p)
				if err != nil {
					return fmt.Errorf("custom schema: %v", err)
				}
			default:
				return fmt.Errorf("unknown parameter property for %#v: %#v",
					name, t)
			}
		}
	}

	return nil
}

// Convert a struct field to JSON schema.
func fieldToSchema(prog *Program, fName, tagName string, ref Reference, f *ast.Field) (*Schema, error) {
	var p Schema

	if f.Doc != nil {
		p.Description = f.Doc.Text()
	} else if f.Comment != nil {
		p.Description = f.Comment.Text()
	}
	p.Description = strings.TrimSpace(p.Description)

	var tags []string
	p.Description, tags = parseTags(p.Description)
	err := setTags(fName, ref.File, &p, tags)
	if err != nil {
		return nil, err
	}

	// Don't need to carry on if we're loading our own schema.
	if p.CustomSchema != "" {
		return &p, nil
	}

	pkg := ref.Package
	var name *ast.Ident

	dbg("fieldToSchema: %v", f.Names)

	sw := f.Type
start:
	switch typ := sw.(type) {

	// Interface, only useful for its description.
	case *ast.InterfaceType:
		if len(f.Names) == 0 {
			return nil, fmt.Errorf("field has no Names: %#v", f)
		}

		field := f.Names[0].Obj.Decl.(*ast.Field)
		switch typ := field.Type.(type) {
		case *ast.SelectorExpr:
			pkgSel, ok := typ.X.(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("typ.X is not ast.Ident: %#v", typ.X)
			}
			pkg = pkgSel.Name
			name = typ.Sel

			lookup := pkg + "." + name.Name
			if _, err := GetReference(prog, ref.Context, false, lookup, ref.File); err != nil {
				return nil, fmt.Errorf("GetReference: %v", err)
			}
		case *ast.Ident:
			name = typ
		}

	// Pointer type; we don't really care about this for now, so just read over
	// it.
	case *ast.StarExpr:
		sw = typ.X
		goto start

	// Simple identifiers such as "string", "int", "MyType", etc.
	case *ast.Ident:
		mappedType, mappedFormat := MapType(prog, pkg+"."+typ.Name)
		if mappedType == "" {
			// Only check for canonicalType if this isn't mapped.
			canon, err := canonicalType(ref.File, pkg, typ)
			if err != nil {
				return nil, fmt.Errorf("cannot get canonical type: %v", err)
			}
			if canon != nil {
				sw = canon
				goto start
			}
		}
		if mappedType != "" {
			p.Type = JSONSchemaType(mappedType)
		} else {
			p.Type = JSONSchemaType(typ.Name)
		}
		if mappedFormat != "" {
			p.Format = mappedFormat
		}

		// e.g. string, int64, etc.: don't need to look up.
		if isPrimitive(p.Type) {
			return &p, nil
		}

		p.Type = ""
		name = typ

	// Anonymous struct
	case *ast.StructType:
		p.Type = "object"
		p.Properties = map[string]*Schema{}
		for _, f := range typ.Fields.List {
			propName := goutil.TagName(f, tagName)
			prop, err := fieldToSchema(prog, propName, tagName, ref, f)
			if err != nil {
				return nil, fmt.Errorf("anon struct: %v", err)
			}

			p.Properties[propName] = prop
		}

	// An expression followed by a selector, e.g. "pkg.foo"
	case *ast.SelectorExpr:
		pkgSel, ok := typ.X.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("typ.X is not ast.Ident: %#v", typ.X)
		}

		pkg = pkgSel.Name
		name = typ.Sel

		lookup := pkg + "." + name.Name
		t, f := MapType(prog, lookup)
		if t == "" {
			// Only check for canonicalType if this isn't mapped.
			canon, err := canonicalType(ref.File, pkgSel.Name, typ.Sel)
			if err != nil {
				return nil, fmt.Errorf("cannot get canonical type: %v", err)
			}
			if canon != nil {
				sw = canon
				goto start
			}
		}

		p.Format = f
		if t != "" {
			p.Type = JSONSchemaType(t)
			return &p, nil
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

			return &p, nil
		}

	// Maps
	case *ast.MapType:
		// As far as I can find there is no obvious/elegant way to represent
		// this in JSON schema, so it's just an object.
		p.Type = "object"
		vtyp, vpkg, err := findTypeIdent(typ.Value, pkg)
		if err != nil {
			// we cannot find a mapping to a concrete type,
			// so we cannot define the type of the maps -> ?
			dbg("ERR FOUND MapType: %s", err.Error())
			return &p, nil
		}
		if isPrimitive(vtyp.Name) {
			// we are done, no need for a lookup of a custom type
			p.AdditionalProperties = &Schema{Type: JSONSchemaType(vtyp.Name)}
			return &p, nil
		}

		_, lref, err := lookupTypeAndRef(ref.File, vpkg, vtyp.Name)
		if err == nil {
			// found additional properties
			p.AdditionalProperties = &Schema{Reference: lref}
			// Make sure the reference is added to `prog.References`:
			_, err := GetReference(prog, ref.Context, false, lref, ref.File)
			if err != nil {
				dbg("ERR, Could not find additionalProperties Reference: %s", err.Error())
			}
		} else {
			dbg("ERR, Could not find additionalProperties: %s", err.Error())
		}
		return &p, nil

	// Array and slices.
	case *ast.ArrayType:
		p.Type = "array"

		err := resolveArray(prog, ref, pkg, &p, typ.Elt)
		if err != nil {
			return nil, err
		}

		return &p, nil

	default:
		return nil, fmt.Errorf("fieldToSchema: unknown type: %T", typ)
	}

	if name == nil {
		return &p, nil
	}

	// Check if the type resolves to a Go primitive.
	lookup := pkg + "." + name.Name
	t, err := getTypeInfo(prog, lookup, ref.File)
	if err != nil {
		return nil, err
	}
	if t != "" {
		p.Type = t
		if isPrimitive(p.Type) {
			return &p, nil
		}
	}

	if i := strings.LastIndex(lookup, "/"); i > -1 {
		lookup = pkg[i+1:] + "." + name.Name
	}

	p.Description = "" // SwaggerHub will complain if both Description and $ref are set.
	p.Reference = lookup

	return &p, nil
}

func dropTypePointers(typ ast.Expr) ast.Expr {
	var t *ast.StarExpr
	var ok bool
	for t, ok = typ.(*ast.StarExpr); ok; t, ok = typ.(*ast.StarExpr) {
		typ = t.X
	}
	return typ
}

func findTypeIdent(typ ast.Expr, curPkg string) (*ast.Ident, string, error) {
	typ = dropTypePointers(typ)
	if i, ok := typ.(*ast.Ident); ok {
		// after droping the stars we have the ident:
		return i, curPkg, nil
	}

	se, ok := typ.(*ast.SelectorExpr)
	if !ok {
		// not ident, not a package selector expr, cannot find ident
		return nil, "", fmt.Errorf("fieldTypeIdent: cannot find ident for type: %T", typ)
	}

	pkgSel, ok := se.X.(*ast.Ident)
	if !ok {
		return nil, "", fmt.Errorf("fieldTypeIdent: SelectorExpr's typ.X is not ast.Ident: %#v", se.X)
	}
	return se.Sel, pkgSel.Name, nil
}

func lookupTypeAndRef(file, pkg, name string) (string, string, error) {
	// Check if the type resolves to a Go primitive.
	lookup := pkg + "." + name
	ts, _, _, err := findType(file, pkg, name)
	if err != nil {
		return "", "", err
	}
	/*
		ident, ok := ts.Name.(*ast.Ident)
		if !ok {
			// in getTypeInfo, there is a special case for anonymous stucts ?
			// but I am not sure how can we be looking up for an anonymous struct
			// O_o
			return "", "", fmt.Errorf("could not find ident for file: %s, pkg: %s, name: %s",
				file, pkg, name)
		}
		t := JSONSchemaType(ident.Name)
	*/
	t := JSONSchemaType(ts.Name.Name)

	sRef := lookup
	if i := strings.LastIndex(pkg, "/"); i > -1 {
		sRef = pkg[i+1:] + "." + name
	}
	return t, sRef, nil
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

		dbg("resolveArray: ident: %#v in %#v", typ.Name, pkg)

		p.Items = &Schema{Type: JSONSchemaType(typ.Name)}

		// Map []byte to []string.
		if typ.Name == "byte" {
			p.Items = nil
			p.Type = "string"
			return nil
		}

		// Only list primitives as type.
		if isPrimitive(p.Items.Type) {
			return nil
		}

		// Rest is assumed to be a custom type, and references with $ref after
		// the switch.
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

	sRef := lookup
	if i := strings.LastIndex(pkg, "/"); i > -1 {
		sRef = pkg[i+1:] + "." + name.Name
	}
	p.Items = &Schema{Reference: sRef}

	// Add to prog.References if not there already
	rName, rPkg := ParseLookup(lookup, ref.File)

	if _, ok := prog.References[filepath.Base(rPkg)+"."+rName]; !ok {
		_, err = GetReference(prog, ref.Context, false, lookup, ref.File)
	}
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
	"uint":    "integer",
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
	// TODO: REMOVE THE prog PARAM, as this function is not
	// using it anymore.
	dbg("getTypeInfo: %#v in %#v", lookup, filePath)
	name, pkg := ParseLookup(lookup, filePath)

	// Find type.
	ts, _, _, err := findType(filePath, pkg, name)
	if err != nil {
		return "", err
	}

	// TODO: This is *ast.StructType in cases for anonymous structs.
	// fmt.Printf("%T, %v -> %v -> %#v\n", ts.Type, ts.Type, ok, ident)
	ident, ok := ts.Type.(*ast.Ident)
	if !ok {
		return "", nil
	}

	t := JSONSchemaType(ident.Name)
	return t, nil
}

// Get the canonical type.
func canonicalType(currentFile, pkgPath string, typ *ast.Ident) (ast.Expr, error) {
	if goutil.PredeclaredType(typ.Name) {
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

func readAndUnmarshalSchemaFile(path string, target interface{}) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file %q: %v", path, err)
	}

	var f func([]byte, interface{}) error
	switch strings.ToLower(filepath.Ext(path)) {
	default:
		return fmt.Errorf("unknown file type: %q", path)
	case ".json":
		f = json.Unmarshal
	case ".yaml":
		f = yaml.Unmarshal
	}
	if err := f(data, target); err != nil {
		return fmt.Errorf("unmarshal schema: %q: %v", path, err)
	}
	return nil
}
