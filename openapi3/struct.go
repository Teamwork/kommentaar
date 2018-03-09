package openapi3

import (
	"errors"
	"fmt"
	"go/ast"
)

// Convert an ast.Field to the OpenAPI format.
//
// TODO: Make this not suck.
func typeString(f *ast.Field) (string, error) {
	switch typ := f.Type.(type) {

	// Don't support interface{} for now. We'd have to add a lot of complexity
	// for it, and not sure if we're ever going to need it.
	case *ast.InterfaceType:
		return "", errors.New("interface{} is not supported")

	// "string", "int", "MyType", etc.
	case *ast.Ident:
		return typ.Name, nil

	// []<..>
	case *ast.ArrayType:
		elt, ok := typ.Elt.(*ast.Ident)
		if !ok {
			var elt ast.Expr
			s := ""

			star, ok := typ.Elt.(*ast.StarExpr)
			if ok {
				// e.g. "[]*models.Language"
				s = "*"
				elt = star.X
			} else {
				// e.g. "[]models.Language"
				elt = typ.Elt
			}

			slt, ok := elt.(*ast.SelectorExpr)
			if !ok {
				return "", fmt.Errorf("can't type assert %T %[1]v", typ.Elt)
			}

			xid, ok := slt.X.(*ast.Ident)
			if !ok {
				return "", fmt.Errorf("can't type assert selector %T %[1]v", slt.X)
			}

			return fmt.Sprintf("[]%v%v.%v", s, xid.Name, slt.Sel.Name), nil
		}

		// e.g. "[]Foo"
		return "[]" + elt.Name, nil

	// pkg.<..>
	case *ast.SelectorExpr:
		// e.g. "models.Language".
		return resolveSelectorExpr(typ)

	// *<..>
	//
	case *ast.StarExpr:
		// e.g. "*models.Session"
		xid, ok := typ.X.(*ast.SelectorExpr)
		if !ok {
			iid, ok := typ.X.(*ast.Ident)
			if ok {
				return "*" + iid.Name, nil
			}

			return "", fmt.Errorf("can't type assert selector 1 %T %[1]v", typ.X)
		}

		xid2, ok := xid.X.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("can't type assert selector 2 %T %[1]v", xid.X)
		}

		return fmt.Sprintf("*%v.%v", xid2.Name, xid.Sel.Name), nil

	// map<..>
	case *ast.MapType:
		key, ok := typ.Key.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("can't type assert key %T %[1]v", typ.Key)
		}

		// TODO: just to get ValidationError working..
		val, ok := typ.Value.(*ast.ArrayType)
		if !ok {
			return "", fmt.Errorf("can't type assert value %T %[1]v", typ.Key)
		}

		valIdent, ok := val.Elt.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("can't type assert value %T %[1]v", typ.Key)
		}

		return fmt.Sprintf("map[%v][]%v", key.Name, valIdent.Name), nil

	default:
		return "", fmt.Errorf("unknown type: %T", typ)
	}
}

func resolveSelectorExpr(sel *ast.SelectorExpr) (string, error) {
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", fmt.Errorf("can't type assert pkg selector %T %[1]v", sel.X)
	}

	return fmt.Sprintf("%v.%v", pkg.Name, sel.Sel.Name), nil
}
