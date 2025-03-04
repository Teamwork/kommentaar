package docparse

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/teamwork/utils/v2/goutil"
	"github.com/teamwork/utils/v2/sliceutil"
	yaml "gopkg.in/yaml.v3"
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

	FieldWhitelist []string `json:"field-whitelist,omitempty" yaml:"field-whitelist,omitempty"`

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
			fmt.Fprintf(os.Stderr, "empty `%s` tag for %s. tags value: %s\n", tagName, schema.Title, p.KindField.Tag.Value)
			name = p.Name
		}

		prop, err := fieldToSchema(prog, name, tagName, ref, p.KindField, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %v: %v", ref.Lookup, err)
		}

		if !sliceutil.Contains([]string{"path", "query", "form"}, ref.Context) {
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
	paramEnum      = "enum"
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
		case paramEnum:
			// For this type of enum, we figure out the variations based on the type.
			p.Type = "enum"

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
				t = strings.ReplaceAll(t[5:], "\n", " ")
				for _, e := range strings.Split(t, " ") {
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
			case strings.HasPrefix(t, "field-whitelist: "):
				for _, e := range strings.Split(t[17:], " ") {
					e = strings.TrimSpace(e)
					if e != "" {
						p.FieldWhitelist = append(p.FieldWhitelist, e)
					}
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
func fieldToSchema(
	prog *Program,
	fName, tagName string,
	ref Reference,
	f *ast.Field,
	generics map[string]string,
) (*Schema, error) {
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
		if mappedType != "" {
			p.Type = JSONSchemaType(mappedType)
		}
		if generics != nil && generics[typ.Name] != "" {
			mappedType = "generics"
			p.Type = JSONSchemaType(generics[typ.Name])
		}
		if p.Type == "enum" && len(p.Enum) == 0 {
			if variations, err := getEnumVariations(ref.File, pkg, typ.Name); len(variations) > 0 {
				p.Enum = variations
			} else if err != nil {
				return nil, err
			}
		}
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
			prop, err := fieldToSchema(prog, propName, tagName, ref, f, generics)
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
		ts, _, importPath, err := findType(ref.File, pkg, name.Name)
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(importPath, pkg) { // import alias
			pkg = importPath
		}

		switch resolvType := ts.Type.(type) {
		case *ast.ArrayType:
			isEnum := p.Type == "enum"
			p.Type = "array"
			err := resolveArray(prog, ref, pkg, &p, resolvType.Elt, isEnum, generics)
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
		if generics != nil && generics[vtyp.Name] != "" {
			vtyp.Name = generics[vtyp.Name]
		}
		if isPrimitive(vtyp.Name) {
			// we are done, no need for a lookup of a custom type
			if vtyp.Name != "any" {
				p.AdditionalProperties = &Schema{Type: JSONSchemaType(vtyp.Name)}
			}
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
		isEnum := p.Type == "enum"
		p.Type = "array"

		err := resolveArray(prog, ref, pkg, &p, typ.Elt, isEnum, generics)
		if err != nil {
			return nil, err
		}

		return &p, nil

	// Generic types
	case *ast.IndexExpr:
		genericsIdent, ok := typ.X.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("unknown generic type: %T", typ.X)
		}
		if err := fillGenericsSchema(prog, &p, tagName, ref, genericsIdent, generics, typ.Index); err != nil {
			return nil, fmt.Errorf("generic fieldToSchema: %v", err)
		}
		return &p, nil

	case *ast.IndexListExpr:
		genericsIdent, ok := typ.X.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("unknown generic type: %T", typ.X)
		}
		if err := fillGenericsSchema(prog, &p, tagName, ref, genericsIdent, generics, typ.Indices...); err != nil {
			return nil, fmt.Errorf("generic fieldToSchema: %v", err)
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

// fillGenericsSchema fills the schema with the generic type information. As the
// types can be different for every generics declaration they will need to be a
// anonymos object in the schema output instead of a reusable reference.
func fillGenericsSchema(
	prog *Program,
	p *Schema,
	tagName string,
	ref Reference,
	genericsIdent *ast.Ident,
	generics map[string]string,
	indices ...ast.Expr,
) error {
	genericsType, _, _, err := findType(ref.File, ref.Package, genericsIdent.Name)
	if err != nil {
		return fmt.Errorf("cannot find generic type: %v", err)
	}

	var genericsTemplateIDs []string
	for _, item := range genericsType.TypeParams.List {
		for _, name := range item.Names {
			genericsTemplateIDs = append(genericsTemplateIDs, name.Name)
		}
	}

	if generics == nil {
		generics = make(map[string]string)
	}
	if len(genericsTemplateIDs) > 0 {
		if len(indices) != len(genericsTemplateIDs) {
			return fmt.Errorf("generic type has %d template IDs, but %d arguments were provided",
				len(genericsTemplateIDs), len(indices))
		}
		for i := 0; i < len(indices); i++ {
			arg, _, err := findTypeIdent(indices[i], ref.Package)
			if err != nil {
				return fmt.Errorf("cannot find generic type argument: %v", err)
			}
			generics[genericsTemplateIDs[i]] = arg.Name
		}
	}

	genericsStruct, ok := genericsType.Type.(*ast.StructType)
	if !ok {
		return fmt.Errorf("generic type is not a struct: %T", genericsType.Type)
	}

	p.Type = "object"
	if p.Properties == nil {
		p.Properties = make(map[string]*Schema)
	}

	for _, field := range genericsStruct.Fields.List {
		fieldName := goutil.TagName(field, tagName)
		schema, err := fieldToSchema(prog, fieldName, tagName, ref, field, generics)
		if err != nil {
			return fmt.Errorf("generic fieldToSchema: %v", err)
		}
		p.Properties[fieldName] = schema
	}

	return nil
}

// Helper function to extract enum variations from a file.
func getEnumVariations(currentFile, pkgPath, typeName string) ([]string, error) {
	resolvedPath, pkg, err := resolvePackage(currentFile, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("could not resolve package: %v", err)
	}
	decls, err := getDecls(pkg, resolvedPath)
	if err != nil {
		return nil, err
	}
	var variations []string
	for _, decl := range decls {
		if decl.vs == nil {
			continue
		}
		if exprToString(decl.vs.Type) != typeName {
			continue
		}
		if len(decl.vs.Names) == 0 || len(decl.vs.Values) == 0 {
			continue
		}
		// All enums variations are required to have the type as their prefix.
		if !strings.HasPrefix(exprToString(decl.vs.Names[0]), typeName) {
			continue
		}
		variations = append(variations, exprToString(decl.vs.Values[0]))
	}

	return variations, nil
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
	t := JSONSchemaType(ts.Name.Name)

	sRef := lookup
	if i := strings.LastIndex(pkg, "/"); i > -1 {
		sRef = pkg[i+1:] + "." + name
	}
	return t, sRef, nil
}

func resolveArray(
	prog *Program,
	ref Reference,
	pkg string,
	p *Schema,
	typ ast.Expr,
	isEnum bool,
	generics map[string]string,
) error {
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

		if generics != nil && generics[typ.Name] != "" {
			p.Items = &Schema{Type: JSONSchemaType(generics[typ.Name])}
		} else {
			p.Items = &Schema{Type: JSONSchemaType(typ.Name)}
		}

		// Generally an item is an enum rather than the array itself
		if len(p.Enum) > 0 {
			p.Items.Enum = p.Enum
			p.Enum = nil
		}

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

		// handle import aliases
		_, _, importPath, err := findType(ref.File, pkg, name.Name)
		if err != nil {
			return fmt.Errorf("resolveArray: findType: %v", err)
		}
		if !strings.HasSuffix(importPath, pkg) {
			pkg = importPath
		}

	case *ast.MapType:
		p.Items = &Schema{Type: "object"}
		return nil

	default:
		return fmt.Errorf("fieldToSchema: unknown array type: %T", typ)
	}

	// Check if the type resolves to a Go primitive.
	lookup := pkg + "." + name.Name
	t, err := getTypeInfo(prog, lookup, ref.File)
	if err != nil {
		return err
	}
	if t != "" && isPrimitive(t) {
		if p.Items == nil {
			p.Items = &Schema{}
		}
		p.Items.Type = t
		if isEnum && len(p.Items.Enum) == 0 {
			if variations, err := getEnumVariations(ref.File, pkg, name.Name); len(variations) > 0 {
				p.Items.Enum = variations
			} else if err != nil {
				return err
			}
		}
		return nil
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
	//"null", "boolean", "object", "array", "number", "string", "integer", "any"
	return sliceutil.Contains([]string{
		"null", "boolean", "number", "string", "integer", "any",
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

func getTypeInfo(_ *Program, lookup, filePath string) (string, error) {
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
	data, err := os.ReadFile(path)
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
