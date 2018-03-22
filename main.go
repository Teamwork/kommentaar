package main // import "github.com/teamwork/kommentaar"

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"arp242.net/sconfig"
	"github.com/kr/pretty"
	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/kommentaar/html"
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
	out := flag.String("out", "openapi3-yaml", `output function. Valid values are "openapi3-yaml", "openapi3-json", and
"openapi3-jsonindent" for OpenAPI3 output as YAML, JSON, or indented
JSON respectively, and "html" for HTML documentation.`)

	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// TODO: allow setting the default outFunc from the config file as well.
	var outFunc func(io.Writer, *docparse.Program) error
	switch strings.ToLower(*out) {
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

	if *config != "" {
		err := sconfig.Parse(&prog.Config, *config, nil)
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
