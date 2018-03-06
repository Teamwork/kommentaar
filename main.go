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
	out := flag.String("out", "openapi3", `output function. Valid values are "openapi" for OpenAPI3 JSON output
and "dump" to show the intermediate internal representation (useful
for development)`)

	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var outFunc func(io.Writer, docparse.Program) error
	switch strings.ToLower(*out) {
	case "dump":
		outFunc = dumpOut
	case "openapi3":
		outFunc = openapi3.Write
	default:
		fmt.Fprintf(os.Stderr, "invalid value for -out: %#v\n\n", *out)
		flag.Usage()
	}

	docparse.InitProgram()

	err := sconfig.Parse(&docparse.Prog.Config, "./config", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}

	err = docparse.FindComments(paths, outFunc)
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
