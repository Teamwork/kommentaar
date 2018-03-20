package openapi3

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"

	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/utils/sliceutil"
)

// Convert a struct to a JSON schema.
func structToSchema(name string, ref docparse.Reference) (*Schema, error) {
	schema := &Schema{
		Title:       name,
		Description: ref.Info,
		Properties:  map[string]*Schema{},
	}

	for _, p := range ref.Params {
		if p.KindField == nil {
			return nil, fmt.Errorf("p.KindField is nil for %v", name)
		}

		name := docparse.JSONTag(p.KindField)
		if name == "-" {
			continue
		}
		if name == "" {
			name = p.Name
		}

		prop, err := fieldToSchema(ref, p.KindField)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %v: %v", ref.Lookup, err)
		}

		if p.Required {
			schema.Required = append(schema.Required, name)
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

// Convert a struct field to JSON schema.
func fieldToSchema(ref docparse.Reference, f *ast.Field) (*Schema, error) {
	var p Schema

	// TODO: parse {..} tags from here. That should probably be in docparse
	// though(?)
	if f.Doc != nil {
		p.Description = f.Doc.Text()
	} else if f.Comment != nil {
		p.Description = f.Comment.Text()
	}
	p.Description = strings.TrimSpace(p.Description)

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
		p.Type = typ.Name
		if k, ok := kindMap[p.Type]; ok {
			p.Type = k
		}

		// e.g. string, int64, etc.: don't need to look up as struct.
		if isPrimitive(p.Type) {
			//fmt.Println("PRIM", f.Names, f.Type)
			return &p, nil
		}

		// TODO: won't work if this points at array:
		//
		//   type foo struct { bar foo }
		//   type bar []int64

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

		lookup := pkg + "." + name.Name
		if l, ok := docparse.MapTypes[lookup]; ok {
			p.Type = l
			if k, ok := kindMap[p.Type]; ok {
				p.Type = k
			}

			return &p, nil
		}

		// Deal with array.
		// TODO: don't do this inline but at the end. Reason it doesn't work not
		// is because we always use GetReference().
		ts, _, _, err := docparse.FindType(ref.File, pkg, name.Name)
		if err != nil {
			return nil, err
		}

		switch resolvType := ts.Type.(type) {
		case *ast.ArrayType:
			p.Type = "array"
			err := resolveArray(ref, &p, resolvType.Elt)
			if err != nil {
				return nil, err
			}

			return &p, nil
		}

		//return &p, nil

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

		err := resolveArray(ref, &p, typ.Elt)
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

	lookup := pkg + "." + name.Name
	_, err := docparse.GetReference(lookup, ref.File)
	if err != nil {
		return nil, fmt.Errorf("getReference: %v", err)
	}

	if i := strings.LastIndex(lookup, "/"); i > -1 {
		lookup = pkg[i+1:] + "." + name.Name
	}

	p.Reference = "#/components/schemas/" + lookup

	return &p, nil
}

func resolveArray(ref docparse.Reference, p *Schema, typ ast.Expr) error {
	asw := typ

	pkg := ref.Package
	var name *ast.Ident

arrayStart:
	switch typ := asw.(type) {

	// Ignore *
	case *ast.StarExpr:
		asw = typ.X
		goto arrayStart

	// Simple identifier
	case *ast.Ident:

		dbg("resolveArray: ident: %#v", typ.Name)

		p.Items = &Schema{Type: typ.Name}
		if k, ok := kindMap[p.Type]; ok {
			p.Items.Type = k
		}

		if isPrimitive(typ.Name) {
			p.Items.Type = typ.Name
			return nil
		}

		if typ.Name == "byte" {
			p.Items = nil
			p.Type = "string"
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

	lookup := pkg + "." + name.Name
	_, err := docparse.GetReference(lookup, ref.File) // TODO: just as sanity check
	if err != nil {
		return fmt.Errorf("resolveArray: GetReference error for %v: %v",
			lookup, err)
	}

	if i := strings.LastIndex(pkg, "/"); i > -1 {
		lookup = pkg[i+1:] + "." + name.Name
	}
	p.Items = &Schema{
		Reference: "#/components/schemas/" + lookup,
	}

	return nil
}

func isPrimitive(n string) bool {
	//"null", "boolean", "object", "array", "number", "string", "integer",
	return sliceutil.InStringSlice([]string{
		"null", "boolean", "number", "string", "integer",
	}, n)
}
