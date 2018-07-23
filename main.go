package main // import "github.com/teamwork/kommentaar"

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/kr/pretty"
	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/kommentaar/html"
	"github.com/teamwork/kommentaar/kconfig"
	"github.com/teamwork/kommentaar/openapi2"
)

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "usage: kommentaar [pkg pkg...]\n\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	config := flag.String("config", "", "configuration file")
	debug := flag.Bool("debug", false, "print debug output to stderr")
	addr := flag.String("serve", "", "serve HTML output on this address, instead of writing to\n"+
		"stdout; every page load will rescan the source tree")
	out := flag.String("out", "openapi2-yaml", `output function, valid values are:
	openapi2-yaml        OpenAPI/Swagger 2.0 as YAML
	openapi2-json        OpenAPI/Swagger 2.0 as JSON
	openapi2-jsonindent  OpenAPI/Swagger 2.0 as JSON indented
	html                 HTML documentation
`)
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "could not create CPU profile: %v\n", err)
			os.Exit(10)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "could not start CPU profile: %v\n", err)
			os.Exit(10)
		}
		defer pprof.StopCPUProfile()
	}

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
	case "html":
		if *addr != "" {
			outFunc = html.ServeHTML(*addr)
		} else {
			outFunc = html.WriteHTML
		}

	// These are just for debugging/testing.
	case "ls":
		outFunc = lsAll
	default:
		_, _ = fmt.Fprintf(os.Stderr, "invalid value for -out: %#v\n\n", *out)
		flag.Usage()
	}

	prog := docparse.NewProgram(*debug)

	if *config != "" {
		err := kconfig.Load(prog, *config)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, err.Error()+"\n")
			os.Exit(1)
		}
	}

	prog.Config.Paths = paths
	prog.Config.Output = outFunc

	err := docparse.FindComments(os.Stdout, prog)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "could not create memory profile: %v\n", err)
			os.Exit(10)
		}
		runtime.GC()
		err = pprof.WriteHeapProfile(f)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "could not write memory profile: %v\n", err)
			os.Exit(10)
		}
		_ = f.Close()
	}
}

func lsAll(_ io.Writer, prog *docparse.Program) error {
	_, err := pretty.Print(prog)
	fmt.Println("")
	return err
}
