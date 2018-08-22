// Package kconfig loads the configuration for Kommentaar.
package kconfig

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"arp242.net/sconfig"
	_ "arp242.net/sconfig/handlers/html/template" // template.HTML handler
	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/kommentaar/html"
	"github.com/teamwork/kommentaar/openapi2"
)

// Load the configuration.
func Load(prog *docparse.Program, file string) error {
	err := sconfig.Parse(&prog.Config, file, sconfig.Handlers{
		"Output": func(line []string) error {
			if len(line) != 1 {
				return fmt.Errorf("invalid: %q", strings.Join(line, " "))
			}

			var err error
			prog.Config.Output, err = Output(line[0], "")
			return err
		},

		"DefaultResponse": func(line []string) error {
			code, err := strconv.ParseInt(line[0], 10, 32)
			if err != nil {
				return fmt.Errorf("first word must be response code: %v", err)
			}

			// TODO: validate rest as well.
			// TODO: change syntax to allow content-type?

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
		return fmt.Errorf("could not load config: %v", err)
	}

	// Merge in some defaults.
	def := map[string]string{
		// stdlib
		"sql.NullBool":    "bool",
		"sql.NullFloat64": "float64",
		"sql.NullInt64":   "int64",
		"sql.NullString":  "string",
		"time.Time":       "string",

		// github.com/guregu/null
		"null.Bool":   "bool",
		"null.Float":  "float64",
		"null.Int":    "int64",
		"null.String": "string",
		"null.Time":   "string",
		"zero.Bool":   "bool",
		"zero.Float":  "float64",
		"zero.Int":    "int64",
		"zero.String": "string",
		"zero.Time":   "string",
	}
	for k, v := range def {
		if _, ok := prog.Config.MapTypes[k]; !ok {
			prog.Config.MapTypes[k] = v
		}
	}

	def = map[string]string{
		// stdlib
		"time.Time": "date-time",

		// github.com/guregu/null
		"null.Time": "date-time",
		"zero.Time": "date-time",
	}
	for k, v := range def {
		if _, ok := prog.Config.MapFormats[k]; !ok {
			prog.Config.MapFormats[k] = v
		}
	}

	if prog.Config.Output == nil {
		prog.Config.Output = openapi2.WriteYAML
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
