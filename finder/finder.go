// Package finder finds comment blocks in Go packages.
package finder // import "github.com/teamwork/kommentaar/finder"

import (
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"

	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/utils/goutil"
)

// FindComments finds all comments in the given packages.
func FindComments(
	paths []string,
	getDoc func(string, string) (*docparse.Endpoint, error),
	output func(io.Writer, docparse.Program) error,
) error {

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
			for _, f := range pkg.Files {
				for _, c := range f.Comments {
					e, err := getDoc(c.Text(), p.ImportPath)
					if err != nil {
						return err
					}
					if e == nil {
						continue
					}
					docparse.Prog.Endpoints = append(docparse.Prog.Endpoints, e)
				}
			}
		}
		err = output(os.Stdout, docparse.Prog)
		if err != nil {
			return err
		}
	}
	return nil
}
