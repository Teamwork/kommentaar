package docparse

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/teamwork/utils/v2/goutil"
	"github.com/teamwork/utils/v2/sliceutil"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/tools/go/packages"
)

// FindComments finds all comments in the given paths or packages.
func FindComments(w io.Writer, prog *Program) error {
	pkgs, err := goutil.Expand(prog.Config.Packages, packages.NeedName|packages.NeedFiles)
	if err != nil {
		return err
	}

	allErr := []error{}

	for _, pkg := range pkgs {
		// Ignore test package.
		if strings.HasSuffix(pkg.Name, "_test") {
			continue
		}

		for _, fullPath := range pkg.GoFiles {
			// Print as just <pkgname>/<file> in errors instead of full path.
			//printDebug = strings.Contains(fullPath, "/thread_controller.go")
			relPath := pkg.PkgPath + "/" + filepath.Base(fullPath)
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, fullPath, nil, parser.ParseComments)
			if err != nil {
				allErr = append(allErr, err)
				continue
			}

			for _, c := range f.Comments {
				//printDebug = printDebug || strings.Contains(c.Text(), "PATCH /v2/tickets/{ticketId}/messages/{id}.json Messages")
				e, relLine, err := parseComment(prog, c.Text(), pkg.PkgPath, fullPath)
				if err != nil {
					p := fset.Position(c.Pos())
					allErr = append(allErr, fmt.Errorf("%v:%v %v",
						relPath, p.Line+relLine, err))
					continue
				}
				if e == nil || e[0] == nil {
					continue
				}
				e[0].Pos = fset.Position(c.Pos())
				e[0].End = fset.Position(c.End())

				// Copy info from main endpoint to aliases.
				for i, a := range e[1:] {
					s := *e[0]
					e[i+1] = &s
					e[i+1].Path = a.Path
					e[i+1].Method = a.Method
					e[i+1].Tags = a.Tags
				}

				prog.Endpoints = append(prog.Endpoints, e...)
				//printDebug = false
			}
		}
	}

	for i, ref := range prog.References {
		old := ref.Package
		ref.Package = MapPackage(prog, ref.Package)
		delete(prog.References, i)
		i = strings.Replace(i, old, ref.Package, 1)
		if prev, ok := prog.References[i]; ok {
			return fmt.Errorf("name collition: %s.%s <-> %s.%s(%s)", old, ref.Name, prev.Package, prev.Name, prev.File)
		}
		prog.References[i] = ref
		FixSchemaPackage(ref.Schema)
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
	// Note: making this more efficient means http.ServeHTML is also harder.
	return prog.Config.Output(w, prog)
}

type declCache struct {
	ts   *ast.TypeSpec
	vs   *ast.ValueSpec
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
	resolvedPath, pkg, err := resolvePackage(currentFile, pkgPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("could not resolve package: %v", err)
	}
	dbg("findType: file: %#v, pkgPath: %#v, name: %#v resolvedPath:%s pkg:%s", currentFile, pkgPath, name, resolvedPath, pkg.Name)

	decls, err := getDecls(pkg, resolvedPath)
	if err != nil {
		return nil, "", "", err
	}

	for _, ts := range decls {
		if ts.ts == nil {
			continue
		}

		if ts.ts.Name.Name == name {
			dbg("findType: declaration file:%s importpath: %s ts.name:%s name:%s", ts.file, pkg.ImportPath, ts.ts.Name.Name, name)
			// impPath := pkg.ImportPath
			// if impPath == "." {
			// 	impPath = pkg.Name
			// }
			return ts.ts, ts.file, pkg.ImportPath, nil
		}
	}

	if name == "pathParams" {
		debug.PrintStack()
	}
	return nil, "", "", fmt.Errorf("could not find type %#v in package %#v:%s",
		name, resolvedPath, pkg.ImportPath)
}

func findValue(currentFile, pkgPath, name string) (
	vs *ast.ValueSpec,
	filePath string,
	importPath string,
	err error,
) {
	dbg("findValue: file: %#v, pkgPath: %#v, name: %#v", currentFile, pkgPath, name)
	resolvedPath, pkg, err := resolvePackage(currentFile, pkgPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("could not resolve package: %v", err)
	}

	decls, err := getDecls(pkg, resolvedPath)
	if err != nil {
		return nil, "", "", err
	}

	for _, decl := range decls {
		if decl.vs == nil {
			continue
		}
		for _, ident := range decl.vs.Names {
			if ident.Name == name {
				impPath := pkg.ImportPath
				if impPath == "." {
					impPath = pkg.Name
				}
				return decl.vs, decl.file, impPath, nil
			}
		}
	}

	return nil, "", "", fmt.Errorf("could not find value %#v in package %#v",
		name, resolvedPath)
}

func resolvePackage(currentFile, pkgPath string) (
	resolvedPath string, pkg *build.Package, err error,
) {
	resolvedPath = pkgPath
	pkg, err = goutil.ResolvePackage(pkgPath, 0)
	if err != nil && currentFile != "" {
		resolved, resolveErr := goutil.ResolveImport(currentFile, pkgPath)
		if resolveErr != nil {
			return "", nil, resolveErr
		}
		if resolved != "" {
			resolvedPath = resolved
			pkg, err = goutil.ResolvePackage(resolvedPath, 0)
		}
	}
	if err != nil {
		return "", nil, err
	}
	return resolvedPath, pkg, nil
}

func getDecls(pkg *build.Package, pkgPath string) ([]declCache, error) {
	// Try to load from cache.
	decls, ok := declsCache[pkg.ImportPath]
	if ok {
		return decls, nil
	}

	dbg("getDecls: parsing dir %#v: %#v", pkg.Dir, pkg.GoFiles)
	fset := token.NewFileSet()
	pkgs, err := goutil.ParseFiles(fset, pkg.Dir, pkg.GoFiles, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}

	for _, p := range pkgs {
		for path, f := range p.Files {
			for _, d := range f.Decls {
				// Only need to cache *ast.GenDecl with what we're interested in.
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

							decls = append(decls, declCache{
								ts: ts, file: path,
							})
						}

						// Constants or variables, used for printing.
						if vs, ok := s.(*ast.ValueSpec); ok {
							decls = append(decls, declCache{
								vs: vs, file: path,
							})
						}
					}
				}
			}
		}
	}

	declsCache[pkg.ImportPath] = decls
	return decls, nil
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
//	type Foo struct {
//	  Field Bar
//	}
//
// A GetReference("Foo", "") call will add two entries to prog.References: Foo
// and Bar (but only Foo is returned).
func GetReference(prog *Program, context string, isEmbed bool, lookup, sourceFilePath string) (*Reference, error) {
	wrapper := ""
	isSlice := false
	if strings.HasPrefix(lookup, "[") && strings.HasSuffix(lookup, "]") && strings.Contains(lookup, ":") {
		wrapper = strings.TrimPrefix(strings.Split(lookup, ":")[0], "[")
		lookup = strings.TrimSuffix(strings.Split(lookup, ":")[1], "]")
	}

	if strings.HasPrefix(lookup, "[") && string(lookup[1]) == "]" {
		isSlice = true
		lookup = lookup[2:]
	}

	dbg("getReference: lookup: %#v -> filepath: %#v", lookup, sourceFilePath)
	refName, pkgAlias := ParseLookup(lookup, sourceFilePath)
	dbg("getReference: pkg: %#v -> name: %#v", pkgAlias, refName)

	// Find type.
	ts, refFilePath, refPkg, err := findType(sourceFilePath, pkgAlias, refName)
	if err != nil {
		return nil, err
	}

	// Already parsed this one, don't need to do it again.
	if ref, ok := prog.References[Reference{Name: refName, Package: refPkg}.String()]; ok {
		// Update context: some structs are embedded but also referenced
		// directly.
		if ref.IsEmbed {
			prog.References[ref.String()] = ref
		}
		return &ref, nil
	}

	var st *ast.StructType
	switch typ := ts.Type.(type) {
	case *ast.StructType:
		st = typ
	case *ast.InterfaceType:
		// dummy StructType, we'll just be using the doc from the interface.
		st = &ast.StructType{Fields: &ast.FieldList{}}
	case *ast.ArrayType:
		arLookup := fmt.Sprintf("[]%v.%v", strings.Split(lookup, ".")[0], exprToString(typ.Elt))
		if wrapper != "" {
			arLookup = fmt.Sprintf("[%v:%v]", wrapper, arLookup)
		}
		return GetReference(prog, context, isEmbed, arLookup, sourceFilePath)
	default:
		return nil, ErrNotStruct{ts, fmt.Sprintf(
			"%v is not a struct or interface but a %T", refName, ts.Type)}
	}

	ref := Reference{
		Name:    refName,
		Package: refPkg,
		// NOTE: Is it needed? there's a lookup param which i think it's the same
		// Lookup looks like the way a variable is defined in the doc
		// i.e. Request body: >>>> threadupdate.Message <<<< this
		//Lookup:  filepath.Base(refPkg) + "." + refName,
		Lookup:  lookup,
		File:    refFilePath,
		Context: context,
		IsEmbed: isEmbed,
		IsSlice: isSlice,
	}

	prog.References[ref.String()] = ref

	if ts.Doc != nil {
		ref.Info = strings.TrimSpace(ts.Doc.Text())
	}
	if wrapper != "" {
		ref.Wrapper = wrapper
	}

	var tagName string
	switch ref.Context {
	case ctxPath, ctxQuery, ctxForm:
		tagName = ref.Context
	case ctxReq, ctxResp:
		tagName = prog.Config.StructTag
	default:
		return nil, fmt.Errorf("invalid context: %q", context)
	}

	// Parse all the fields.
	// TODO(param): only reason we do this is to make things a bit easier during
	// refactor. We should pass st to structToSchema() or something.
	for _, f := range st.Fields.List {
		if f.Comment != nil && ref.Context != ctxPath {
			if hasTag(f.Comment.Text(), paramOmitDoc) {
				continue
			}
		}

		if len(f.Names) == 0 {
			// Skip embedded structs without tags; we merge them later.
			if f.Tag == nil {
				continue
			}

			var (
				ref *Reference
				err error
			)
			switch t := f.Type.(type) {
			case *ast.Ident:
				ref, err = resolveType(prog, context, false, t, refFilePath, refPkg)
			case *ast.StarExpr:
				ex, _ := t.X.(*ast.Ident)
				ref, err = resolveType(prog, context, false, ex, refFilePath, refPkg)
			}

			if err != nil {
				return nil, fmt.Errorf("could not lookup %s in %s: %s",
					err, f.Type, lookup)
			}
			lookup = ref.String()
		}

		// Names is an array in cases like "Foo, Bar string".
		for _, fName := range f.Names {
			if !fName.IsExported() {
				if f.Tag != nil {
					tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`")).Get(tagName)
					if tag != "" {
						return nil, fmt.Errorf("not exported but has %q tag: %s.%s field %v",
							tagName, refPkg, refName, f.Names)
					}
				}

				continue
			}
			p := Param{
				Name:      fName.Name,
				KindField: f,
			}
			ref.Fields = append(ref.Fields, p)
		}
	}

	prog.References[ref.String()] = ref
	var (
		nested       []string
		nestedTagged []*ast.Field
	)

	// Scan all fields of f if it refers to a struct. Do this after storing the
	// reference in prog.References to prevent cyclic lookup issues.
	for _, f := range st.Fields.List {
		if f.Comment != nil && ref.Context != ctxPath {
			if hasTag(f.Comment.Text(), paramOmitDoc) {
				continue
			}
		}

		var isEmbed bool
		if len(f.Names) == 0 {
			isEmbed = true
		}

		if goutil.TagName(f, tagName) == "-" {
			continue
		}
		if !isEmbed {
			exp := false
			for _, fName := range f.Names {
				if fName.IsExported() {
					exp = true
					break
				}
			}
			if !exp {
				continue
			}
		}

		nestLookup, err := findNested(prog, context, isEmbed, f, refFilePath, refPkg)
		if err != nil {
			return nil, fmt.Errorf("\n  findNested: %v", err)
		}
		if isEmbed {
			if f.Tag == nil {
				nested = append(nested, nestLookup)
			} else if len(f.Names) == 0 {
				nestedTagged = append(nestedTagged, f)
			}
		}
	}

	// Add in embedded structs with a tag.
	for _, n := range nestedTagged {
		ename := cases.Title(language.English).String(goutil.TagName(n, tagName))
		n.Names = []*ast.Ident{{
			Name: ename,
		}}
		ref.Fields = append(ref.Fields, Param{
			Name:      ename,
			KindField: n,
		})
	}

	// Convert to JSON Schema.
	schema, err := structToSchema(prog, refName, tagName, ref)
	if err != nil {
		return nil, fmt.Errorf("%v can not be converted to JSON schema: %v", refName, err)
	}
	ref.Schema = schema

	changed := false

	for _, p := range ref.Schema.Properties {
		// Check if any fields are whitelisted, if not continue onto next property
		if len(p.FieldWhitelist) == 0 {
			continue
		}

		changed = true

		// Get the package so we can lookup the correct reference
		split := strings.Split(p.Reference, ".")
		lookupStruct := strings.Join(split[:len(split)-1], ".")
		if lookupStruct != "" {
			lookupStruct = lookupStruct + "."
		}

		for i, f := range ref.Fields {
			if lookupStruct+f.Name != p.Reference {
				continue
			}

			// Find the referenced struct
			reference, err := GetReference(prog, context, false, lookupStruct+f.Name, sourceFilePath)
			if err != nil {
				return nil, fmt.Errorf("could not get referenced struct %s", lookupStruct+f.Name)
			}

			fields := []*ast.Field{}
			for _, field := range reference.Fields {
				if sliceutil.Contains(p.FieldWhitelist, strings.ToLower(field.Name)) {
					fields = append(fields, field.KindField)
				}
			}

			// Construct the parameter using the given fields
			ref.Fields[i] = Param{
				Name: f.Name,
				KindField: &ast.Field{
					Doc: &ast.CommentGroup{
						List: []*ast.Comment{{Slash: 0, Text: reference.Schema.Description}},
					},
					Names: f.KindField.Names,
					Type: &ast.StructType{
						Struct: 0,
						Fields: &ast.FieldList{
							Opening: 0,
							List:    fields,
						},
					},
					Tag:     f.KindField.Tag,
					Comment: f.KindField.Comment,
				},
			}
		}
	}

	// If the fields have been changed, regenerate the schema with the new fields
	if changed {
		schema, err = structToSchema(prog, refName, tagName, ref)
		if err != nil {
			return nil, fmt.Errorf("%v can not be converted to JSON schema: %v", refName, err)
		}
		ref.Schema = schema
	}

	// Merge for embedded structs without a tag.
	for _, n := range nested {
		ref.Fields = append(ref.Fields, prog.References[n].Fields...)

		if prog.References[n].Schema != nil {
			for k, v := range prog.References[n].Schema.Properties {
				if _, ok := ref.Schema.Properties[k]; !ok {
					ref.Schema.Properties[k] = v
				}
			}
		}
	}

	if ref.IsSlice {
		sliceSchema := &Schema{
			Type:  "array",
			Items: ref.Schema,
		}
		ref.Schema = sliceSchema
	}

	if ref.Wrapper != "" {
		wrappedSchema := &Schema{
			Title:      ref.Name,
			Type:       "object",
			Properties: map[string]*Schema{},
		}

		wrappedSchema.Properties[ref.Wrapper] = ref.Schema
		ref.Schema = wrappedSchema
	}

	prog.References[ref.String()] = ref

	return &ref, nil
}

func findNested(prog *Program, context string, isEmbed bool, f *ast.Field, filePath, pkg string) (string, error) {
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
		if !goutil.PredeclaredType(typ.Name) {
			name = typ
		}

	// An expression followed by a selector, e.g. "pkg.foo"
	case *ast.SelectorExpr:
		pkgSel, ok := typ.X.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("typ.X is not ast.Ident: %#v", typ.X)
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
			if !goutil.PredeclaredType(elementType.Name) {
				name = elementType
			}

		// "pkg.foo"
		case *ast.SelectorExpr:
			pkgSel, ok := elementType.X.(*ast.Ident)
			if !ok {
				return "", fmt.Errorf("elementType.X is not ast.Ident: %#v",
					elementType.X)
			}
			name = elementType.Sel
			pkg = pkgSel.Name
		}

	case *ast.MapType:
		msw := typ.Value

	mapStart: // I feel dirty doing this...  :/
		switch elementType := msw.(type) {

		// Ignore *
		case *ast.StarExpr:
			msw = elementType.X
			goto mapStart

		// Simple identifier
		case *ast.Ident:
			if !goutil.PredeclaredType(elementType.Name) {
				name = elementType
			}

		// "pkg.foo"
		case *ast.SelectorExpr:
			pkgSel, ok := elementType.X.(*ast.Ident)
			if !ok {
				return "", fmt.Errorf("elementType.X is not ast.Ident: %#v",
					elementType.X)
			}
			name = elementType.Sel
			pkg = pkgSel.Name
		}
	}

	if name == nil {
		return "", nil
	}

	lookup := pkg + "." + name.Name

	// if i := strings.LastIndex(pkg, "/"); i > -1 {
	// 	lookup = pkg[i+1:] + "." + name.Name
	// }

	// Don't need to add stuff we map to Go primitives.
	if x, _ := MapType(prog, lookup); x != "" {
		return lookup, nil
	}
	if _, ok := prog.References[lookup]; !ok {
		ref, err := resolveType(prog, context, isEmbed, name, filePath, pkg)
		if err != nil {
			return "", fmt.Errorf("%v.%v: %v", pkg, name, err)
		}

		if ref != nil {
			return ref.String(), nil
		}
	}
	return lookup, nil
}

// Add the type declaration to references.
func resolveType(prog *Program, context string, isEmbed bool, typ *ast.Ident, filePath, pkg string) (*Reference, error) {
	var ts *ast.TypeSpec
	if typ.Obj == nil {
		var err error
		ts, _, _, err = findType(filePath, pkg, typ.Name)
		if err != nil {
			return nil, err
		}
	} else {
		var ok bool
		ts, ok = typ.Obj.Decl.(*ast.TypeSpec)
		if !ok {
			return nil, fmt.Errorf("resolveType: not a type declaration but %T",
				typ.Obj.Decl)
		}
	}

	// Only need to add struct types. Stuff like "type Foo string" gets added as
	// simply a string.
	_, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil, nil
	}

	// This sets prog.References
	lookup := pkg + "." + typ.Name
	ref, err := GetReference(prog, context, isEmbed, lookup, filePath)
	fmt.Fprintf(os.Stderr, "=====resolved %s -> %s\n", lookup, ref.String())

	return ref, err
}

// ParseLookup for the package and name, if lookup is an imported path e.g
// models.Foo then:
// pkg: models, name: Foo
// in the case of current package the filePath is used, e.g:
// pkg: Dir(filePath), name: Foofunc ParseLookup(lookup string, filePath string) (name, pkg string) {
func ParseLookup(lookup string, filePath string) (name, pkg string) {
	if c := strings.LastIndex(lookup, "."); c > -1 {
		// imported path: models.Foo
		return lookup[c+1:], lookup[:c]
	}

	// Current package: Foo
	return lookup, path.Dir(filePath)
}
