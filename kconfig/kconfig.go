// Package kconfig loads the configuration for Kommentaar.
package kconfig // import "github.com/teamwork/kommentaar/kconfig"

import (
	"fmt"
	"io"
	"strings"

	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/kommentaar/html"
	"github.com/teamwork/kommentaar/openapi2"
	"github.com/teamwork/utils/v2/goutil"
	"zgo.at/sconfig"
	_ "zgo.at/sconfig/handlers/html/template" // template.HTML handler
)

// Load the configuration.
func Load(prog *docparse.Program, file string) error {
	err := sconfig.Parse(&prog.Config, file, sconfig.Handlers{
		"Output": func(line []string) error {
			if len(line) != 1 {
				return fmt.Errorf("invalid output: %q", strings.Join(line, " "))
			}

			var err error
			prog.Config.Output, err = Output(line[0], "")
			return err
		},

		"DefaultResponse": func(line []string) error {
			if prog.Config.DefaultResponse == nil {
				prog.Config.DefaultResponse = make(map[int]docparse.Response)
			}

			code, resp, err := docparse.ParseResponse(prog, "", "Response "+strings.Join(line, " "))
			if err != nil {
				return err
			}
			if resp == nil {
				return fmt.Errorf("malformed default response: %q", strings.Join(line, " "))
			}

			if _, ok := prog.Config.DefaultResponse[code]; ok {
				return fmt.Errorf("default response code %v defined more than once", code)
			}

			prog.Config.DefaultResponse[code] = *resp
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("could not load config: %v", err)
	}

	// Validate that MapType is a Go primitive.
	for k, v := range prog.Config.MapTypes {
		if !goutil.PredeclaredType(v) {
			return fmt.Errorf("map-type '%s %s' is not a predeclared type", k, v)
		}
	}

	// Set a default output.
	if prog.Config.Output == nil {
		prog.Config.Output = openapi2.WriteYAML
	}

	if prog.Config.StructTag == "" {
		prog.Config.StructTag = "json"
	}

	return nil
}

// Output gets the output function from a string.
func Output(out, addr string) (func(io.Writer, *docparse.Program) error, error) {
	var outFunc func(io.Writer, *docparse.Program) error
	switch strings.ToLower(out) {
	case "openapi2-yaml":
		outFunc = openapi2.WriteYAML
	case "openapi2-json":
		outFunc = openapi2.WriteJSON
	case "openapi2-jsonindent":
		outFunc = openapi2.WriteJSONIndent
	case "html":
		if addr != "" {
			outFunc = html.ServeHTML(addr)
		} else {
			outFunc = html.WriteHTML
		}
	default:
		return nil, fmt.Errorf("unknown value: %q", out)
	}

	return outFunc, nil
}
