package docparse

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strings"

	"github.com/teamwork/utils/goutil"
)

// FindComments finds all comments in the given paths or packages.
func FindComments(paths []string, output func(io.Writer, Program) error) error {
	pkgPaths, err := goutil.Expand(paths, build.FindOnly)
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
				if i := strings.Index(relPath, p.ImportPath); i > -1 {
					relPath = relPath[i:]
				}

				for _, c := range f.Comments {
					e, err := ParseComment(c.Text(), p.ImportPath, fullPath)
					if err != nil {
						allErr = append(allErr, fmt.Errorf("%v: %v", relPath, err))
						continue
					}
					if e == nil {
						continue
					}

					//p := fset.Position(c.Pos())
					//e.Location = fmt.Sprintf("%v:%v:%v", path, p.Line, p.Column)
					e.Pos = fset.Position(c.Pos())
					e.End = fset.Position(c.End())
					Prog.Endpoints = append(Prog.Endpoints, e)
				}
			}
		}
	}

	if len(allErr) > 0 {
		fmt.Fprintf(os.Stderr, "%v errors\n", len(allErr))
		for _, err := range allErr {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}

	// TODO: it's probably better to call this per package or file, rather than
	// once for everything (much more memory-efficient for large packages).
	// OTOH, perhaps this is "good enough"?
	return output(os.Stdout, Prog)
}

var declsCache = make(map[string][]*ast.TypeSpec)

// FindType attempts to find a type.
//
// currentFile is the current file being parsed.
//
// pkgPath is the package path of the tyope you want to find. It can either be a
// fully qualified path (i.e. "github.com/user/pkg") or a package from the
// currentPkg imports (i.e. "models" will resolve to "github.com/desk/models" if
// that is imported in currentPkg).
func FindType(currentFile, pkgPath, name string) (*ast.TypeSpec, error) {
	dbg("FindType: %#v %#v %#v", currentFile, pkgPath, name)

	pkg, err := goutil.ResolvePackage(pkgPath, 0)
	if err != nil && currentFile != "" {
		resolved, resolveErr := goutil.ResolveImport(currentFile, pkgPath)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if resolved != "" {
			pkgPath = resolved
			pkg, err = goutil.ResolvePackage(pkgPath, 0)
		}
	}
	if err != nil {
		return nil, err
	}

	// Try to load from cache.
	decls, ok := declsCache[pkgPath]
	if !ok {
		fset := token.NewFileSet()
		dbg("FindType: parsing dir %#v: %#v", pkg.Dir, pkg.GoFiles)
		pkgs, err := goutil.ParseFiles(fset, pkg.Dir, pkg.GoFiles, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		// TODO: we should probably support this, or at least the common case of
		// "pkg" and "pkg_test".
		if len(pkgs) != 1 {
			return nil, fmt.Errorf("more than one package in %v", pkgPath)
		}

		for _, p := range pkgs {
			for _, f := range p.Files {
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

								decls = append(decls, ts)
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
		if ts.Name.Name == name {
			return ts, nil
		}
	}

	return nil, fmt.Errorf("could not find %v in %v", name, pkgPath)
}
