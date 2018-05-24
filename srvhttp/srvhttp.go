// Package srvhttp contains HTTP handlers for serving Kommentaar documentation.
package srvhttp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/kommentaar/html"
	"github.com/teamwork/kommentaar/kconfig"
	"github.com/teamwork/kommentaar/openapi2"
)

// Args for the HTTP handlers.
type Args struct {
	Paths    []string // Paths to scan.
	Config   string   // Kommentaar config file.
	NoScan   bool     // Don't scan the paths, but instead load and output one of the *File.
	YAMLFile string
	JSONFile string
	HTMLFile string
}

// YAML outputs as OpenAPI2 YAML.
func YAML(args Args) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := run(args, openapi2.WriteYAML, args.YAMLFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, out)
	}
}

// JSON outputs as OpenAPI2 JSON.
//
// Set the "indented" query parameter to get formatted JSON.
func JSON(args Args) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var f func(io.Writer, *docparse.Program) error
		if r.URL.Query().Get("indented") != "" {
			f = openapi2.WriteJSONIndent
		} else {
			f = openapi2.WriteJSON
		}

		out, err := run(args, f, args.JSONFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, out)
	}
}

// HTML outputs as HTML documentation.
func HTML(args Args) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := run(args, html.WriteHTML, args.HTMLFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error: %v", err)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, out)
	}
}

func run(
	args Args,
	out func(io.Writer, *docparse.Program) error,
	file string,
) (string, error) {

	if args.NoScan {
		o, err := ioutil.ReadFile(file)
		return string(o), err
	}

	prog := docparse.NewProgram(false)
	if args.Config != "" {
		err := kconfig.Load(prog, args.Config)
		if err != nil {
			return "", err
		}
	}

	prog.Config.Paths = args.Paths
	prog.Config.Output = out

	buf := bytes.NewBuffer(nil)
	err := docparse.FindComments(buf, prog)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
