package main

import (
	"flag"
	"fmt"
	"go/build"
	"go/parser"
	"os"

	"github.com/teamwork/kommentaar/parse"
	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/loader"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: kommentaar [pkg]\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()
	paths := flag.Args()

	if len(paths) != 1 {
		fmt.Fprintf(os.Stderr, "need exactly one package\n")
		flag.Usage()
	}

	err := process(paths[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
}

func process(path string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	ctx := &build.Default
	pkg, err := buildutil.ContainingPackage(ctx, cwd, path)
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

	endpoints, err := findEndpoints(prog)
	if err != nil {
		return err
	}

	fmt.Println(endpoints)

	//err = toOpenAPI3(endpoints)
	//if err != nil {
	//	return err
	//}

	return nil
}

func findEndpoints(prog *loader.Program) ([]parse.Endpoint, error) {
	endpoints := []parse.Endpoint{}

	for _, p := range prog.InitialPackages() {
		for _, f := range p.Files {
			for _, c := range f.Comments {
				if c == nil {
					continue
				}

				e, err := parse.Parse(c.Text())
				if err != nil {
					return nil, err
				}
				if e == nil {
					continue
				}

				endpoints = append(endpoints, *e)
			}
		}
	}

	return endpoints, nil
}
