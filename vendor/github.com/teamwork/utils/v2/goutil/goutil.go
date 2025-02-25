// Package goutil provides functions to work with Go source files.
//
//nolint:staticcheck
package goutil // import "github.com/teamwork/utils/v2/goutil"

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Expand a list of package and/or directory names to Go package names.
//
//   - "./example" is expanded to "full/package/path/example".
//   - "/absolute/src/package/path" is abbreviated to "package/path".
//   - "full/package" is kept-as is.
//   - "package/path/..." will include "package/path" and all subpackages.
//
// The packages will be sorted with duplicate packages removed. The /vendor/
// directory is automatically ignored.
func Expand(paths []string, mode packages.LoadMode) ([]*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: mode}, paths...)
	if err != nil {
		return nil, err
	}

	out := make([]*packages.Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return nil, pkg.Errors[0]
		}
		out = append(out, pkg)
	}

	if len(out) == 0 {
		return nil, errors.New("cannot find package")
	}

	sort.Slice(out, func(i, j int) bool { return out[i].PkgPath < out[j].PkgPath })

	return out, nil
}

var cwd string

// ResolvePackage resolves a package path, which can either be a local directory
// relative to the current dir (e.g. "./example"), a full path (e.g.
// ~/go/src/example"), or a package path (e.g. "example").
func ResolvePackage(path string, mode build.ImportMode) (pkg *build.Package, err error) {
	if len(path) == 0 {
		// TODO: maybe resolve like '.'? Dunno what makes more sense.
		return nil, errors.New("cannot resolve empty string")
	}

	switch path[0] {
	case '/':
		pkg, err = build.ImportDir(path, mode)
	case '.':
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		pkg, err = build.ImportDir(path, mode)
	default:
		if cwd == "" {
			cwd, err = os.Getwd()
			if err != nil {
				return nil, err
			}
		}
		pkg, err = build.Import(path, cwd, mode)
	}
	if err != nil {
		return nil, err
	}

	return pkg, err
}

// ResolveWildcard finds all subpackages in the "example/..." format. The
// "/vendor/" directory will be ignored.
func ResolveWildcard(path string, mode build.ImportMode) ([]*build.Package, error) {
	root, err := ResolvePackage(path[:len(path)-4], mode)
	if err != nil {
		return nil, err
	}

	// Gather a list of directories with *.go files.
	goDirs := make(map[string]struct{})
	err = filepath.Walk(root.Dir, func(path string, info os.FileInfo, _ error) error {
		if !strings.HasSuffix(path, ".go") || info.IsDir() || strings.Contains(path, "/vendor/") {
			return nil
		}

		goDirs[filepath.Dir(path)] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var out []*build.Package
	for d := range goDirs {
		pkg, err := ResolvePackage(d, mode)
		if err != nil {
			return nil, err
		}
		out = append(out, pkg)
	}

	return out, nil
}

// ParseFiles parses the given list of *.go files.
//
// The advantage of this over parser.ParseDir() is that you can use the result
// of ResolvePackage() as input, which avoids a directory scan and takes build
// tags in to account (ParseDir() ignores any build tags).
func ParseFiles(
	fset *token.FileSet,
	dir string,
	files []string,
	mode parser.Mode,
) (map[string]*ast.Package, error) {

	pkgs := make(map[string]*ast.Package)
	var firstErr error

	for _, file := range files {
		path := filepath.Join(dir, "/", file)

		src, err := parser.ParseFile(fset, path, nil, mode)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		name := src.Name.Name
		pkg, found := pkgs[name]
		if !found {
			pkg = &ast.Package{
				Name:  name,
				Files: make(map[string]*ast.File),
			}
			pkgs[name] = pkg
		}
		pkg.Files[path] = src
	}

	return pkgs, firstErr
}

var importsCache = make(map[string]map[string]string)

// ResolveImport resolves an import name (e.g. "models") to the full imported
// package (e.g. "github.com/teamwork/desk/models") for a file. An empty string
// is returned if the package can't be resolved.
//
// This will automatically keep a cache with name -> packagePath mappings to
// avoid having to parse the file more than once.
func ResolveImport(file, pkgName string) (string, error) {
	imports, ok := importsCache[file]
	if !ok {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			return "", err
		}

		imports = make(map[string]string)
		for _, imp := range f.Imports {
			var base string
			p := strings.Trim(imp.Path.Value, `"`)
			if imp.Name != nil {
				base = imp.Name.Name
			} else {
				base = path.Base(p)
			}

			imports[base] = p
		}

		importsCache[file] = imports
	}

	r, ok := imports[pkgName]
	if !ok {
		currentPkg := path.Base(path.Dir(file))
		if pkgName == currentPkg {
			r = "."
		}
	}
	return r, nil
}

// TagName gets the tag name for a struct field with all attributes (like
// omitempty) removed. It will return the struct field name if there is no tag.
//
// This function does not do any validation on the tag format. Use "go vet"!
func TagName(f *ast.Field, n string) string {
	// For e.g.:
	//  A, B string `json:"x"`
	//
	// Most (all?) marshallers and such will simply skip this anyway as
	// duplicate keys usually doesn't make too much sense.
	if len(f.Names) > 1 {
		panic(fmt.Sprintf("cannot use TagName on struct with more than one name: %v",
			f.Names))
	}

	if f.Tag == nil {
		if len(f.Names) == 0 {
			return getEmbedName(f.Type)
		}
		return f.Names[0].Name
	}

	tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`")).Get(n)
	if tag == "" {
		if len(f.Names) == 0 {
			return getEmbedName(f.Type)
		}
		return f.Names[0].Name
	}

	if p := strings.Index(tag, ","); p != -1 {
		return tag[:p]
	}
	return tag
}

// Embedded struct:
//
//	Foo `json:"foo"`
func getEmbedName(f ast.Expr) string {
start:
	switch t := f.(type) {
	case *ast.StarExpr:
		f = t.X
		goto start
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.IndexExpr:
		f = t.X
		goto start
	case *ast.IndexListExpr:
		f = t.X
		goto start
	default:
		panic(fmt.Sprintf("can't get name for %#v", f))
	}
}

// PredeclaredType reports if a type is a predeclared built-in type.
//
// Note that this excludes composite types, such as maps, slices, channels, etc.
//
// https://golang.org/ref/spec#Predeclared_identifiers
func PredeclaredType(n string) bool {
	switch n {
	case "any", "bool", "byte", "comparable", "complex64", "complex128", "error", "float32",
		"float64", "int", "int8", "int16", "int32", "int64", "rune", "string",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	default:
		return false
	}
}
