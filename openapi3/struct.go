package openapi3

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"

	"github.com/kr/pretty"
	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/utils/sliceutil"
)

var primitives = []string{
	"null", "boolean", "object", "array", "number", "string", "integer",
}

func structToSchema(name string, ref docparse.Reference) (*Schema, error) {
	schema := &Schema{
		Properties:  map[string]SchemaProperty{},
		Title:       name,
		Description: ref.Info,
	}

	for _, p := range ref.Params {
		var prop *SchemaProperty
		if p.KindField != nil {
			var err error
			prop, err = fieldToProperty(ref, p.KindField)
			if err != nil {
				return nil, fmt.Errorf("cannot parse %v: %v", ref.Lookup, err)
			}
		}

		if p.Required {
			// TODO: shouldn't use p.Name here, but the `json` or `yaml` tag.
			schema.Required = append(schema.Required, p.Name)
		}

		// TODO: shouldn't use p.Name here, but the `json` or `yaml` tag.
		schema.Properties[p.Name] = *prop
	}

	return schema, nil
}

// Convert a struct field to JSON schema property.
// https://tools.ietf.org/html/draft-handrews-json-schema-validation-00
func fieldToProperty(ref docparse.Reference, f *ast.Field) (*SchemaProperty, error) {
	var p SchemaProperty

	// TODO: parse {..} tags from here. That should probably be in docparse
	// though(?)
	if f.Doc != nil {
		p.Description = f.Doc.Text()
	} else if f.Comment != nil {
		p.Description = f.Comment.Text()
	}
	p.Description = strings.TrimSpace(p.Description)

	sw := f.Type
start:
	switch typ := sw.(type) {

	// Don't support interface{} for now. We'd have to add a lot of complexity
	// for it, and not sure if we're ever going to need it.
	case *ast.InterfaceType:
		return nil, errors.New("fieldToProperty: interface{} is not supported")

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

		// Custom types.
		if !sliceutil.InStringSlice(primitives, p.Type) {
			err := resolveType(ref, &p, typ)
			if err != nil {
				return nil, err
			}
		}

		return &p, nil

	// An expression followed by a selector, e.g. "pkg.foo"
	case *ast.SelectorExpr:
		err := resolveType(ref, &p, typ.Sel)
		if err != nil {
			return nil, err
		}
		return &p, nil

	// Array and slices.
	case *ast.ArrayType:
		p.Type = "array"

		err := resolveArray(ref, &p, typ.Elt)
		if err != nil {
			return nil, err
		}
		return &p, nil

	// Maps
	case *ast.MapType:
		// As far as I can find there is no obvious/elegant way to represent
		// this in JSON schema, so simply don't support it for now. I don't
		// think we actually use this anywhere.
		// TODO: We should really support this...
		return nil, errors.New("fieldToProperty: maps are not supported due to JSON schema limitations")

	default:
		return nil, fmt.Errorf("fieldToProperty: unknown type: %T", typ)
	}
}

func resolveArray(ref docparse.Reference, p *SchemaProperty, typ ast.Expr) error {
	asw := typ

arrayStart:
	switch elementType := asw.(type) {

	// Ignore *
	case *ast.StarExpr:
		asw = elementType.X
		goto arrayStart

	// Simple identifier
	case *ast.Ident:
		if elementType.Name != "byte" {
			err := resolveType(ref, p, elementType)
			if err != nil {
				return err
			}

			p.Items = Schema{Type: elementType.Name}
		}

		if k, ok := kindMap[elementType.Name]; ok {
			p.Type = k
		}
		return nil

	// "pkg.foo"
	case *ast.SelectorExpr:
		err := resolveType(ref, p, elementType.Sel)
		if err != nil {
			return err
		}
		p.Items = Schema{Type: elementType.Sel.Name}
		return nil

	default:
		return fmt.Errorf("fieldToProperty: unknown array type: %T", elementType)
	}
}

func resolveType(ref docparse.Reference, p *SchemaProperty, typ *ast.Ident) error {

	if typ.Obj == nil {
		// TODO: this seems nil in cases of "pkg.Foo". Not sure how to fix this?
		//return fmt.Errorf("Obj is nil in %#v", typ)
		return nil
	}

	ts, ok := typ.Obj.Decl.(*ast.TypeSpec)
	if !ok {
		return fmt.Errorf("fieldToProperty: not a type declaration but %T", typ.Obj.Decl)
	}

	var nestedP SchemaProperty
	switch nestedTyp := ts.Type.(type) {

	// This refers to a custom name for a primitive (e.g. "type foo
	// string"). Simply set the Type as whatever it refers to.
	case *ast.Ident:
		nestedP.Type = nestedTyp.Name
		if k, ok := kindMap[nestedP.Type]; ok {
			nestedP.Type = k
		}

		p.Type = nestedP.Type

		//if sliceutil.InStringSlice(primitives, nestedP.Type) {
		//	p.Type = nestedP.Type
		//}

		return nil

	// For a struct we want to add the struct as its own componenent and
	// then add a reference to that.
	//  3. Or if it's a struct, set p.Properties to fieldSchema(..)
	case *ast.StructType:
		p.Type = "object"

		lookup := ts.Name.Name
		if !strings.Contains(lookup, ".") {
			lookup = ref.Package + "." + lookup
		}

		// This sets the Prog.References global.
		// TODO: this shouldn't really be here. docparse should find all
		// nested structs and add them as references already.
		nestedRef, err := docparse.GetReference(lookup, "")
		if err != nil {
			_, _ = pretty.Print(ref)
			fmt.Printf("\n\n")
			return fmt.Errorf("fieldToProperty: nested getReference: %v", err)
		}

		nestedSchema, err := structToSchema(nestedRef.Name, *nestedRef)
		if err != nil {
			return fmt.Errorf("fieldToProperty: %v", nestedTyp)
		}

		// TODO: I don't know why these are different types. I think both should
		// be Schema?
		p.Properties = map[string]Schema{}
		for k, v := range nestedSchema.Properties {
			p.Properties[k] = Schema{
				Description: v.Description,
				Type:        v.Type,
			}
		}

		//    "properties": {
		//        "latitude": { "type": "number" },
		//        "longitude": { "type": "number" }
		//    }
		return nil

	default:
		return fmt.Errorf("fieldToProperty: unknown nested: %T", nestedTyp)
	}
}
