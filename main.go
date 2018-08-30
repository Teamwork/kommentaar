package main // import "github.com/teamwork/kommentaar"

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/kommentaar/kconfig"
	"github.com/teamwork/kommentaar/openapi2"
)

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "usage: kommentaar [pkg pkg...]\n\n")
		flag.PrintDefaults()
		os.Exit(2)
	}

	showUsage, err := start()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "kommentaar: %s\n", err)
		if showUsage {
			flag.Usage()
		}
	}
}

var stdout io.Writer = os.Stdout

func start() (bool, error) {
	config := flag.String("config", "", "configuration file")
	debug := flag.Bool("debug", false, "print debug output to stderr")
	addr := flag.String("serve", "", "serve HTML output on this address, instead of writing to\n"+
		"stdout; every page load will rescan the source tree")
	output := flag.String("output", "", `output function, valid values are:
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
			return false, err
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			return false, err
		}
		defer pprof.StopCPUProfile()
	}

	prog := docparse.NewProgram(*debug)

	if *config != "" {
		err := kconfig.Load(prog, *config)
		if err != nil {
			return false, err
		}
	}

	if prog.Config.Output == nil {
		prog.Config.Output = openapi2.WriteYAML
	}
	if *output != "" {
		var err error
		prog.Config.Output, err = kconfig.Output(*output, *addr)
		if err != nil {
			return true, fmt.Errorf("-output: %s", err)
		}
	}

	pkgs := flag.Args()
	if len(pkgs) > 0 {
		prog.Config.Packages = pkgs
	}
	if len(prog.Config.Packages) == 0 {
		prog.Config.Packages = []string{"."}
	}

	err := docparse.FindComments(stdout, prog)
	if err != nil {
		return false, err
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			return false, fmt.Errorf("could not create memory profile: %v", err)
		}
		runtime.GC()
		err = pprof.WriteHeapProfile(f)
		if err != nil {
			return false, fmt.Errorf("could not write memory profile: %v", err)
		}
		_ = f.Close()
	}

	return false, nil
}
