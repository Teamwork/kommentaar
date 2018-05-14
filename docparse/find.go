package docparse

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/teamwork/utils/goutil"
)

// FindComments finds all comments in the given paths or packages.
func FindComments(w io.Writer, prog *Program) error {
	pkgPaths, err := goutil.Expand(prog.Config.Paths, build.FindOnly)
	if err != nil {
		return err
	}

	allErr := []error{}
	for _, p := range pkgPaths {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, p.Dir, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		for _, pkg := range pkgs {
			// Ignore test package.
			if strings.HasSuffix(pkg.Name, "_test") {
				continue
			}

			for fullPath, f := range pkg.Files {
				// Print as just <pkgname>/<file> in errors instead of full path.
				relPath := fullPath
				if p.ImportPath == "." {
					x := strings.Split(relPath, "/")
					relPath = x[len(x)-2] + "/" + x[len(x)-1]
				} else {
					if i := strings.Index(relPath, p.ImportPath); i > -1 {
						relPath = relPath[i:]
					}
				}

				for _, c := range f.Comments {
					e, relLine, err := parseComment(prog, c.Text(), p.ImportPath, fullPath)
					if err != nil {
						p := fset.Position(c.Pos())
						allErr = append(allErr, fmt.Errorf("%v:%v %v",
							relPath, p.Line+relLine, err))
						continue
					}
					if e == nil {
						continue
					}

					e.Pos = fset.Position(c.Pos())
					e.End = fset.Position(c.End())
					prog.Endpoints = append(prog.Endpoints, e)
				}
			}
		}
	}

	if len(allErr) > 0 {
		msg := ""
		for _, err := range allErr {
			msg += err.Error() + "\n"
		}
		return fmt.Errorf("%v\n%v errors occurred", msg, len(allErr))
	}

	// Sort endpoints by tags first, then method, and then path.
	key := func(e *Endpoint) string {
		return fmt.Sprintf("%v%v%v", e.Tags, e.Method, e.Path)
	}
	sort.Slice(prog.Endpoints, func(i, j int) bool {
		return key(prog.Endpoints[i]) < key(prog.Endpoints[j])
	})

	// It's probably better to call this per package or file, rather than once
	// for everything (much more memory-efficient for large packages). OTOH,
	// perhaps this is "good enough"?
	return prog.Config.Output(w, prog)
}

type declCache struct {
	ts   *ast.TypeSpec
	file string
}

var declsCache = make(map[string][]declCache)

// findType attempts to find a type.
//
// currentFile is the current file being parsed.
//
// pkgPath is the package path of the type you want to find. It can either be a
// fully qualified path (i.e. "github.com/user/pkg") or a package from the
// currentPkg imports (i.e. "models" will resolve to "github.com/desk/models" if
// that is imported in currentPkg).
func findType(currentFile, pkgPath, name string) (
	ts *ast.TypeSpec,
	filePath string,
	importPath string,
	err error,
) {
	dbg("FindType: file: %#v, pkgPath: %#v, name: %#v", currentFile, pkgPath, name)

	pkg, err := goutil.ResolvePackage(pkgPath, 0)
	if err != nil && currentFile != "" {
		resolved, resolveErr := goutil.ResolveImport(currentFile, pkgPath)
		if resolveErr != nil {
			return nil, "", "", resolveErr
		}
		if resolved != "" {
			pkgPath = resolved
			pkg, err = goutil.ResolvePackage(pkgPath, 0)
		}
	}
	if err != nil {
		return nil, "", "", fmt.Errorf("could not resolve package: %v", err)
	}

	// Try to load from cache.
	decls, ok := declsCache[pkgPath]
	if !ok {
		fset := token.NewFileSet()
		dbg("FindType: parsing dir %#v: %#v", pkg.Dir, pkg.GoFiles)
		pkgs, err := goutil.ParseFiles(fset, pkg.Dir, pkg.GoFiles, parser.ParseComments)
		if err != nil {
			return nil, "", "", fmt.Errorf("parse error: %v", err)
		}

		for _, p := range pkgs {
			for path, f := range p.Files {
				for _, d := range f.Decls {
					// Only need to cache *ast.GenDecl with one *ast.TypeSpec,
					// as we don't care about functions, imports, and what not.
					if gd, ok := d.(*ast.GenDecl); ok {
						for _, s := range gd.Specs {
							if ts, ok := s.(*ast.TypeSpec); ok {
								// For:
								//     // Commment!
								//     type Foo struct{}
								//
								// The "Comment!" is stored on on the
								// GenDecl.Doc, but for:
								//     type (
								//         // Comment!
								//         Foo struct{}
								//     )
								//
								// it's on the TypeSpec.Doc. Makes no sense to
								// me either, but this makes it more consistent,
								// and easier to access since we only care about
								// the TypeSpec.
								if ts.Doc == nil && gd.Doc != nil {
									ts.Doc = gd.Doc
								}

								decls = append(decls, declCache{ts, path})
								break
							}
						}
					}
				}
			}
		}

		declsCache[pkgPath] = decls
	}

	for _, ts := range decls {
		if ts.ts.Name.Name == name {
			impPath := pkg.ImportPath
			if impPath == "." {
				impPath = pkg.Name
			}
			return ts.ts, ts.file, impPath, nil
		}
	}

	return nil, "", "", fmt.Errorf("could not find type %#v in package %#v",
		name, pkgPath)
}

// ErrNotStruct is used when GetReference resolves to something that is not a
// struct.
type ErrNotStruct struct {
	TypeSpec *ast.TypeSpec
	msg      string
}

func (err ErrNotStruct) Error() string {
	return err.msg
}

// GetReference finds a type by name. It can either be in the current path
// ("SomeStruct"), a package path with a type (e.g.
// "example.com/bar.SomeStruct"), or something from an imported package (e.g.
// "models.SomeStruct").
//
// References are stored in prog.References. This also finds and stores all
// nested references, so for:
//
//  type Foo struct {
//    Field Bar
//  }
//
// A GetReference("Foo", "") call will add two entries to prog.References: Foo
// and Bar (but only Foo is returned).
func GetReference(prog *Program, context, lookup, filePath string) (*Reference, error) {
	dbg("getReference: lookup: %#v -> filepath: %#v", lookup, filePath)

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

	dbg("getReference: pkg: %#v -> name: %#v", pkg, name)

	// Already parsed this one, don't need to do it again.
	if ref, ok := prog.References[lookup]; ok {
		return &ref, nil
	}

	// Find type.
	ts, foundPath, pkg, err := findType(filePath, pkg, name)
	if err != nil {
		return nil, err
	}

	// Make sure it's a struct.
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil, ErrNotStruct{ts, fmt.Sprintf(
			"%v is not a struct but a %T", name, ts.Type)}
	}

	ref := Reference{
		Name:    name,
		Package: pkg,
		Lookup:  filepath.Base(pkg) + "." + name,
		File:    foundPath,
		Context: context,
	}
	if ts.Doc != nil {
		ref.Info = strings.TrimSpace(ts.Doc.Text())
	}

	// Parse all the fields.
	// TODO(param): only reason we do this is to make things a bit easier during
	// refactor. We should pass st to structToSchema() or something.
	for _, f := range st.Fields.List {

		// Skip embedded struct for now.
		// TODO: we want to support this eventually.
		if len(f.Names) == 0 {
			continue
			//return nil, fmt.Errorf("embeded struct is not yet supported")
		}

		// Names is an array in cases like "Foo, Bar string".
		for _, fName := range f.Names {
			p := Param{
				Name:      fName.Name,
				KindField: f,
			}
			ref.Fields = append(ref.Fields, p)
		}
	}

	prog.References[ref.Lookup] = ref

	// Scan all fields of f if it refers to a struct. Do this after storing the
	// reference in prog.References to prevent cyclic lookup issues.
	for _, f := range st.Fields.List {
		if goutil.TagName(f, "json") == "-" {
			continue
		}

		err := findNested(prog, context, f, foundPath, pkg)
		if err != nil {
			return nil, fmt.Errorf("\n  findNested: %v", err)
		}
	}

	// Convert to JSON Schema.
	schema, err := structToSchema(prog, name, ref)
	if err != nil {
		return nil, fmt.Errorf("%v can not be converted to JSON schema: %v", name, err)
	}
	ref.Schema = schema
	prog.References[ref.Lookup] = ref

	return &ref, nil
}

func findNested(prog *Program, context string, f *ast.Field, filePath, pkg string) error {
	var name *ast.Ident

	sw := f.Type
start:
	switch typ := sw.(type) {

	// Pointer type; we don't really care about this for now, so just read over
	// it.
	case *ast.StarExpr:
		sw = typ.X
		goto start

	// Simple identifiers such as "string", "int", "MyType", etc.
	case *ast.Ident:
		if !builtInType(typ.Name) {
			name = typ
		}

	// An expression followed by a selector, e.g. "pkg.foo"
	case *ast.SelectorExpr:
		pkgSel, ok := typ.X.(*ast.Ident)
		if !ok {
			return fmt.Errorf("typ.X is not ast.Ident: %#v", typ.X)
		}
		name = typ.Sel
		pkg = pkgSel.Name

	// Array and slices.
	case *ast.ArrayType:
		asw := typ.Elt

	arrayStart:
		switch elementType := asw.(type) {

		// Ignore *
		case *ast.StarExpr:
			asw = elementType.X
			goto arrayStart

		// Simple identifier
		case *ast.Ident:
			if !builtInType(elementType.Name) {
				name = elementType
			}

		// "pkg.foo"
		case *ast.SelectorExpr:
			pkgSel, ok := elementType.X.(*ast.Ident)
			if !ok {
				return fmt.Errorf("elementType.X is not ast.Ident: %#v",
					elementType.X)
			}
			name = elementType.Sel
			pkg = pkgSel.Name
		}
	}

	if name == nil {
		return nil
	}

	lookup := pkg + "." + name.Name
	if i := strings.LastIndex(pkg, "/"); i > -1 {
		lookup = pkg[i+1:] + "." + name.Name
	}

	// Don't need to add stuff we map to Go primitives.
	if _, ok := mapTypes[lookup]; ok {
		return nil
	}

	if _, ok := prog.References[lookup]; !ok {
		err := resolveType(prog, context, name, filePath, pkg)
		if err != nil {
			return fmt.Errorf("%v.%v: %v", pkg, name, err)
		}
	}
	return nil
}

func resolveType(prog *Program, context string, typ *ast.Ident, filePath, pkg string) error {
	var ts *ast.TypeSpec
	if typ.Obj == nil {
		var err error
		ts, _, _, err = findType(filePath, pkg, typ.Name)
		if err != nil {
			return err
		}
	} else {
		var ok bool
		ts, ok = typ.Obj.Decl.(*ast.TypeSpec)
		if !ok {
			return fmt.Errorf("resolveType: not a type declaration but %T",
				typ.Obj.Decl)
		}
	}

	// Only need to add struct types. Stuff like "type Foo string" gets added as
	// simply a string.
	_, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil
	}

	// This sets prog.References
	lookup := pkg + "." + typ.Name
	_, err := GetReference(prog, context, lookup, filePath)
	return err
}
