// Package finder finds comment blocks in Go files and packages.
package finder // import "github.com/teamwork/kommentaar/finder"

import (
	"go/build"
	"go/parser"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/teamwork/kommentaar/docparse"
	"golang.org/x/tools/go/loader"
)

// Find files and packages.
func Find(
	paths []string,
	getDoc func(string) (*docparse.Endpoint, error),
	output func(io.Writer, []*docparse.Endpoint) error,
) error {

	ctx := &build.Default
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	for _, p := range paths {

		var pkg *build.Package

		// TODO: expand ...
		if strings.HasPrefix(p, ".") {
			// e.g. ./example
			pkg, err = ctx.ImportDir(filepath.Join(cwd, "/", p), build.ImportComment)
		} else {
			// e.g. github.com/teamwork/kommentaar/example
			pkg, err = ctx.Import(p, cwd, build.ImportComment)
		}
		if err != nil {
			return err
		}

		conf := &loader.Config{
			Build:      ctx,
			ParserMode: parser.ParseComments,
		}
		conf.ImportWithTests(pkg.ImportPath)
		prog, err := conf.Load()
		if err != nil {
			return err
		}

		endpoints, err := findEndpoints(prog, getDoc)
		if err != nil {
			return err
		}

		err = output(os.Stdout, endpoints)
		if err != nil {
			return err
		}
	}

	return nil
}

func findEndpoints(
	prog *loader.Program,
	getDoc func(string) (*docparse.Endpoint, error),
) ([]*docparse.Endpoint, error) {
	endpoints := []*docparse.Endpoint{}

	for _, p := range prog.InitialPackages() {
		for _, f := range p.Files {
			for _, c := range f.Comments {
				if c == nil {
					continue
				}

				e, err := getDoc(c.Text())
				if err != nil {
					return nil, err
				}
				if e == nil {
					continue
				}

				endpoints = append(endpoints, e)
			}
		}
	}

	return endpoints, nil
}
