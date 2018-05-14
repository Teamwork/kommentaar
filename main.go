package main // import "github.com/teamwork/kommentaar"

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"arp242.net/sconfig"
	"github.com/kr/pretty"
	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/kommentaar/html"
	"github.com/teamwork/kommentaar/openapi2"
	"github.com/teamwork/kommentaar/openapi3"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: kommentaar [pkg pkg...]\n\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	config := flag.String("config", "", "configuration file")
	debug := flag.Bool("debug", false, "print debug output to stderr")
	out := flag.String("out", "openapi3-yaml", `output function, valid values are:
	openapi2-yaml		OpenAPI/Swagger 2.0 as YAML
	openapi2-json		OpenAPI/Swagger 2.0 as JSON
	openapi2-jsonindent	OpenAPI/Swagger 2.0 as JSON indented
	openapi3-yaml		OpenAPI 3.0.1 as YAML
	openapi3-json		OpenAPI 3.0.1 as JSON
	openapi3-jsonindent	OpenAPI 3.0.1 as JSON indented
	html			as HTML documentation
`)

	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// TODO: allow setting the default outFunc from the config file as well.
	var outFunc func(io.Writer, *docparse.Program) error
	switch strings.ToLower(*out) {
	case "openapi2-yaml":
		outFunc = openapi2.WriteYAML
	case "openapi2-json":
		outFunc = openapi2.WriteJSON
	case "openapi2-jsonindent":
		outFunc = openapi2.WriteJSONIndent
	case "openapi3-yaml":
		outFunc = openapi3.WriteYAML
	case "openapi3-json":
		outFunc = openapi3.WriteJSON
	case "openapi3-jsonindent":
		outFunc = openapi3.WriteJSONIndent
	case "html":
		outFunc = html.WriteHTML

	// These are just for debugging/testing.
	case "ls":
		outFunc = lsAll
	case "ls-ref":
		outFunc = lsRef
	default:
		fmt.Fprintf(os.Stderr, "invalid value for -out: %#v\n\n", *out)
		flag.Usage()
	}

	prog := docparse.NewProgram(*debug)

	if strings.HasPrefix(*out, "openapi2") {
		// TODO should this be settable or just moved to docparse to figure out and set?
		prog.Config.SchemaRefPrefix = "#/definitions"
	}

	if *config != "" {
		err := sconfig.Parse(&prog.Config, *config, sconfig.Handlers{
			// TODO: move this to docparse so it can be called more easily.
			"DefaultResponse": func(line []string) error {
				code, err := strconv.ParseInt(line[0], 10, 32)
				if err != nil {
					return fmt.Errorf("first word must be response code: %v", err)
				}

				// TODO: validate rest as well.

				if prog.Config.DefaultResponse == nil {
					prog.Config.DefaultResponse = make(map[int]docparse.DefaultResponse)
				}
				def := docparse.DefaultResponse{
					Lookup:      strings.Replace(strings.Join(line[1:], " "), "$ref: ", "", 1),
					Description: fmt.Sprintf("%v %v", code, http.StatusText(int(code))),
				}
				ref, err := docparse.GetReference(prog, "", def.Lookup, "")
				if err != nil {
					return err
				}

				def.Schema = *ref.Schema

				prog.Config.DefaultResponse[int(code)] = def

				return nil
			},
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error()+"\n")
			os.Exit(1)
		}
	}

	prog.Config.Paths = paths
	prog.Config.Output = outFunc

	err := docparse.FindComments(os.Stdout, prog)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
}

func lsAll(_ io.Writer, prog *docparse.Program) error {
	_, err := pretty.Print(prog)
	fmt.Println("")
	return err
}

func lsRef(_ io.Writer, prog *docparse.Program) error {
	sp := 0
	var refs []docparse.Reference
	for _, ref := range prog.References {
		refs = append(refs, ref)
		if len(ref.Package) > sp {
			sp = len(ref.Package)
		}
	}

	key := func(r docparse.Reference) string { return fmt.Sprintf("%v.%v", r.Package, r.Name) }
	sort.Slice(refs, func(i, j int) bool { return key(refs[i]) < key(refs[j]) })

	for _, ref := range refs {
		fmt.Printf("%v  %v%v\n",
			ref.Package,
			strings.Repeat(" ", sp-len(ref.Package)),
			ref.Name)
	}

	return nil
}
