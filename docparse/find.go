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

	for _, p := range pkgPaths {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, p.Dir, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		if len(pkgs) != 1 {
			return fmt.Errorf("multiple packages in directory %s", p.Dir)
		}

		for _, pkg := range pkgs {
			for path, f := range pkg.Files {
				for _, c := range f.Comments {
					e, err := Parse(c.Text(), p.ImportPath)
					// Print as just <pkgname>/<file> instead of full path.
					if i := strings.Index(path, p.ImportPath); i > -1 {
						path = path[i:]
					}
					if err != nil {
						return fmt.Errorf("%v: %v", path, err)
					}
					if e == nil {
						continue
					}
					p := fset.Position(c.Pos())
					e.Location = fmt.Sprintf("%v:%v:%v", path, p.Line, p.Column)
					Prog.Endpoints = append(Prog.Endpoints, e)
				}
			}
		}
	}

	// TODO: it's probably better to call this per package or file, rather than
	// once for everything (much more memory-efficient for large packages).
	// OTOH, perhaps this is "good enough"?
	return output(os.Stdout, Prog)
}

var declsCache = make(map[string][]*ast.TypeSpec)

// FindType attempts to find a type.
func FindType(pkgPath, name string) (*ast.TypeSpec, error) {
	pkg, err := goutil.ResolvePackage(pkgPath, 0)
	if err != nil {
		return nil, err
	}

	// Try to load from cache.
	decls, ok := declsCache[pkgPath]
	if !ok {
		fset := token.NewFileSet()
		//fmt.Printf("FindType: parsing dir %#v: %#v\n", pkg.Dir, pkg.GoFiles)
		pkgs, err := goutil.ParseFiles(fset, pkg.Dir, pkg.GoFiles, parser.ParseComments)
		if err != nil {
			return nil, err
		}

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
