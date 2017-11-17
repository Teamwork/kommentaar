package main // import "arp242.net/kommentaar"

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"net/http"
	"os"
	"strings"

	"github.com/teamwork/utils/sliceutil"

	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/loader"
)

type request struct {
	contentType string
	query       []param
	body        reference
}

type reference struct {
	obj string
}

type object struct {
}

type response struct {
	contentType string
	body        reference
}

type param struct {
	name     string
	info     string
	kind     string
	required bool
	ref      string
}

type endpoint struct {
	method   string
	path     string
	tagline  string
	info     string
	request  request
	response response
	tags     []string
}

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

	err = toOpenAPI3(endpoints)
	if err != nil {
		return err
	}

	return nil
}

var methods = []string{http.MethodGet, http.MethodHead, http.MethodPost,
	http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect,
	http.MethodOptions, http.MethodTrace}

func findEndpoints(prog *loader.Program) ([]endpoint, error) {
	endpoints := []endpoint{}

	for _, p := range prog.InitialPackages() {
		for _, f := range p.Files {
			for _, c := range f.Comments {
				if c == nil {
					continue
				}

				e, err := makeEndpoint(prog, c.Text())
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

func makeEndpoint(prog *loader.Program, comment string) (*endpoint, error) {
	e := &endpoint{}
	var tags string
	fmt.Sscanf(comment, "%s /%s %s", &e.method, &e.path, &tags)
	if !sliceutil.InStringSlice(methods, e.method) {
		return nil, nil
	}
	e.path = "/" + e.path

	if tags != "" {
		e.tags = strings.Split(tags, " ")
	}

	// Split the blocks.
	info := map[string]string{}
	state := "desc"
	for _, line := range strings.Split(comment, "\n")[1:] {
		switch {
		case line == "Query:":
			state = "query"

		// TODO: Get content-type and status code.
		case strings.HasPrefix(line, "Request body"):
			state = "reqBody"
		case strings.HasPrefix(line, "Response "):
			state = "respBody"

		default:
			info[state] += line + "\n"
		}
	}

	// First line of desc is tagline, rest is info.
	if info["desc"] != "" {
		desc := strings.Split(strings.TrimSpace(info["desc"]), ".!")
		e.tagline = desc[0]
		if len(desc) > 1 {
			e.info = strings.Join(desc[1:], "\n")
		}
	}

	var err error
	if info["query"] != "" {
		if e.request.query, err = parseParams(info["query"]); err != nil {
			return e, nil
		}
	}
	e.request.contentType = "application/json"
	if info["reqBody"] != "" {
		if e.request.body, err = getReference(prog, info["reqBody"]); err != nil {
			return e, nil
		}
	}
	e.response.contentType = "application/json"
	if info["resBody"] != "" {
		if e.response.body, err = getReference(prog, info["resBody"]); err != nil {
			return e, nil
		}
	}

	return e, nil
}

func getReference(prog *loader.Program, text string) (reference, error) {
	text = strings.TrimSpace(text)
	ref := reference{}

	if !strings.HasPrefix(text, "object:") {
		return ref, fmt.Errorf("must be an object reference: %v", text)
	}
	ref.obj = strings.TrimSpace(strings.Split(text, ":")[1])

	for _, p := range prog.InitialPackages() {
		for _, f := range p.Files {
			for _, d := range f.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok {
					continue
				}

				if gd.Doc != nil {
					fmt.Printf("%s\n", gd.Doc.Text())

					for _, s := range gd.Specs {
						for _, f := range s.(*ast.TypeSpec).Type.(*ast.StructType).Fields.List {
							fmt.Printf("%#v - %#v\n", f.Names[0].Name, f.Doc.Text())
						}
					}
				}
			}
		}

	}

	return ref, nil
}

func parseParams(text string) ([]param, error) {
	text = strings.TrimSpace(text)
	params := []param{}
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		line = strings.TrimSpace(line)
		p := param{}

		// Get description
		s := strings.Split(line, ":")
		if len(s) > 1 {
			if paren := strings.Index(s[1], "("); paren > -1 {
				p.info = strings.TrimSpace(s[1][:paren])
				s[0] += s[1][paren:]
			} else {
				p.info = strings.TrimSpace(s[1])
			}
		}

		m := strings.Split(s[0], "(")
		p.name = strings.TrimSpace(m[0])
		if len(m) > 1 {
			opts := strings.Split(strings.TrimSpace(strings.Trim(m[1], ")")), ",")
			for _, o := range opts {
				o = strings.TrimSpace(o)
				switch {
				case o == "required":
					p.required = true
				case strings.HasPrefix(o, "object:"):
					p.ref = strings.TrimSpace(strings.Split(o, ":")[1])
				default:
					// TODO: validate
					p.kind = o
				}
			}
		}

		params = append(params, p)
	}

	return params, nil
}
