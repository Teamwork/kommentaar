package openapi3

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"

	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/utils/goutil"
	"github.com/teamwork/utils/sliceutil"
)

// Convert a struct to a JSON schema.
func structToSchema(prog *docparse.Program, name string, ref docparse.Reference) (*Schema, error) {
	schema := &Schema{
		Title:       name,
		Description: ref.Info,
		Properties:  map[string]*Schema{},
	}

	for _, p := range ref.Params {
		if p.KindField == nil {
			return nil, fmt.Errorf("p.KindField is nil for %v", name)
		}

		// TODO: doesn't have to be json tag; that's just what Desk happens to
		// use. We should get it from Content-Type or some such instead.
		name := goutil.TagName(p.KindField, "json")
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

func setTags(name string, p *Schema, tags []string) error {
	for _, t := range tags {
		switch t {
		case "required":
			p.Required = append(p.Required, name)

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

			default:
				return fmt.Errorf("unknown parameter tag for %#v: %#v",
					name, t)
			}
		}
	}

	return nil
}

// Convert a struct field to JSON schema.
func fieldToSchema(prog *docparse.Program, fName string, ref docparse.Reference, f *ast.Field) (*Schema, error) {
	var p Schema

	// TODO: parse {..} tags from here. That should probably be in docparse
	// though(?)
	if f.Doc != nil {
		p.Description = f.Doc.Text()
	} else if f.Comment != nil {
		p.Description = f.Comment.Text()
	}
	p.Description = strings.TrimSpace(p.Description)

	var tags []string
	p.Description, tags = docparse.ParseParamsTags(p.Description)
	err := setTags(fName, &p, tags)
	if err != nil {
		return nil, err
	}

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
			err := resolveArray(prog, ref, &p, resolvType.Elt)
			if err != nil {
				return nil, err
			}

			return &p, nil
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

		err := resolveArray(prog, ref, &p, typ.Elt)
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
	// TODO: don't use GetReference here; has many side-effects!
	lookup := pkg + "." + name.Name
	_, err = docparse.GetReference(prog, lookup, ref.File)
	if err != nil {
		nsErr, ok := err.(docparse.ErrNotStruct)
		if ok {
			id, ok := nsErr.TypeSpec.Type.(*ast.Ident)
			if ok {
				p.Type = id.Name
				if k, ok := kindMap[p.Type]; ok {
					p.Type = k
				}

				if isPrimitive(p.Type) {
					return &p, nil
				}
			}
		}

		return nil, fmt.Errorf("GetReference error for %v: %v", lookup, err)
	}

	if i := strings.LastIndex(lookup, "/"); i > -1 {
		lookup = pkg[i+1:] + "." + name.Name
	}

	p.Reference = "#/components/schemas/" + lookup

	return &p, nil
}

func resolveArray(prog *docparse.Program, ref docparse.Reference, p *Schema, typ ast.Expr) error {
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
		if k, ok := kindMap[typ.Name]; ok {
			p.Items.Type = k
		}

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
	// TODO: don't use GetReference here; has many side-effects!
	lookup := pkg + "." + name.Name
	_, err := docparse.GetReference(prog, lookup, ref.File)
	if err != nil {
		nsErr, ok := err.(docparse.ErrNotStruct)
		if ok {
			id, ok := nsErr.TypeSpec.Type.(*ast.Ident)
			if ok {
				p.Type = id.Name
				if k, ok := kindMap[p.Type]; ok {
					p.Type = k
				}

				if isPrimitive(p.Type) {
					return nil
				}
			}
		}

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
