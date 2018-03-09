package main // import "github.com/teamwork/kommentaar"

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"arp242.net/sconfig"
	"github.com/kr/pretty"
	"github.com/teamwork/kommentaar/docparse"
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
JSON respectively. Additionally "dump" can be used to show the internal
representation.`)

	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var outFunc func(io.Writer, docparse.Program) error
	switch strings.ToLower(*out) {
	case "dump":
		outFunc = dumpOut
	case "openapi3-yaml":
		outFunc = openapi3.WriteYAML
	case "openapi3-json":
		outFunc = openapi3.WriteJSON
	case "openapi3-jsonindent":
		outFunc = openapi3.WriteJSONIndent
	default:
		fmt.Fprintf(os.Stderr, "invalid value for -out: %#v\n\n", *out)
		flag.Usage()
	}

	docparse.InitProgram(*debug)

	if *config != "" {
		err := sconfig.Parse(&docparse.Prog.Config, *config, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error()+"\n")
			os.Exit(1)
		}
	}

	err := docparse.FindComments(os.Stdout, paths, outFunc)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
}

func dumpOut(_ io.Writer, prog docparse.Program) error {
	_, err := pretty.Print(prog)
	fmt.Println("")
	return err
}
