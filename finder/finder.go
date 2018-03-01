// Package finder finds comment blocks in Go packages.
package finder // import "github.com/teamwork/kommentaar/finder"

import (
	"errors"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/teamwork/kommentaar/docparse"
)

// Find files and packages.
func Find(
	paths []string,
	getDoc func(string, string) (*docparse.Endpoint, error),
	output func(io.Writer, []*docparse.Endpoint) error,
) error {

	pkgPaths, err := Expand(paths)
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

		var endpoints []*docparse.Endpoint
		for _, pkg := range pkgs {
			for _, f := range pkg.Files {
				for _, c := range f.Comments {
					e, err := getDoc(c.Text(), p.ImportPath)
					if err != nil {
						return err
					}
					if e == nil {
						continue
					}
					endpoints = append(endpoints, e)
				}
			}
		}
		err = output(os.Stdout, endpoints)
		if err != nil {
			return err
		}
	}

	return nil
}

// Expand a list of package and/or directory names to Go package names.
//
//  - "./example" is expanded to "full/package/path/example".
//  - "/absolute/src/package/path" is abbreviated to "package/path".
//  - "full/package" is kept-as is.
//  - "package/path/..." will include "package/path" and all subpackages.
//
// The packages will be sorted with duplicate packages removed. The /vendor/
// directory is automatically ignored.
func Expand(paths []string) ([]*build.Package, error) {
	var out []*build.Package
	for _, p := range paths {
		if strings.HasSuffix(p, "/...") {
			subPkgs, err := ResolveWildcard(p)
			if err != nil {
				return nil, err
			}
			out = append(out, subPkgs...)
			continue
		}

		pkg, err := ResolvePackage(p)
		if err != nil {
			return nil, err
		}
		out = append(out, pkg)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ImportPath < out[j].ImportPath })

	// TODO: remove dupes.

	return out, nil
}

// ResolvePackage resolves a package path, which can either be a local directory
// relative to the current dir (e.g. "./example"), a full path (e.g.
// ~/go/src/example"), or a package path (e.g. "example").
func ResolvePackage(path string) (pkg *build.Package, err error) {
	if len(path) == 0 {
		// TODO: maybe resolve like '.'?
		return nil, errors.New("cannot resolve empty string")
	}

	switch path[0] {
	case '/':
		pkg, err = build.ImportDir(path, build.FindOnly)
	case '.':
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		pkg, err = build.ImportDir(path, build.FindOnly)
	default:
		pkg, err = build.Import(path, ".", build.FindOnly)
	}
	if err != nil {
		return nil, err
	}

	return pkg, err
}

// ResolveWildcard finds all subpackages in the "example/..." format. The
// "/vendor/" directory will be ignored.
func ResolveWildcard(path string) ([]*build.Package, error) {
	root, err := ResolvePackage(path[:len(path)-4])
	if err != nil {
		return nil, err
	}

	// Gather a list of directories with *.go files.
	goDirs := make(map[string]struct{})
	err = filepath.Walk(root.Dir, func(path string, info os.FileInfo, err error) error {
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
		pkg, err := ResolvePackage(d)
		if err != nil {
			return nil, err
		}
		out = append(out, pkg)
	}

	return out, nil
}
