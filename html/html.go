// Package html outputs to HTML.
package html // import "github.com/teamwork/kommentaar/html"

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/teamwork/kommentaar/docparse"
	yaml "gopkg.in/yaml.v2"
)

var funcMap = template.FuncMap{
	"add":    func(a, b int) int { return a + b },
	"status": func(c int) string { return http.StatusText(c) },
	"schema": func(in interface{}) string {
		// TODO: link ref?
		d, err := yaml.Marshal(in)
		if err != nil {
			return fmt.Sprintf("yaml.Marshal error: %v", err)
		}
		return string(d)
	},
}

var mainTpl = template.Must(template.New("mainTpl").Funcs(funcMap).Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>{{.Config.Title}} API documentation {{.Config.Version}}</title>

	<style>
		body {
			font: 16px/1.9em sans-serif;
			background-color: #eee;
		}

		a {
			text-decoration: none;
		}

		p, ul {
			margin: 0;
			padding: 0;
		}

		ul {
			margin-left: 2em;
		}

		h3 {
			font-size: 1.5em;
			position: relative;
			margin-top: 1rem;
			margin-bottom: 0;
			padding: .2rem;
			padding-left: .5rem;
			margin-bottom: -1px;
		}

		h3.js-expand {
			cursor: pointer;
		}

		h4 {
			margin: 0;
			font-size: 16px;
		}

		sup {
			color: #aaa;
		}

		.permalink {
			font-weight: normal;
			color: rgb(0, 0, 238);

			/* Make it a bit easier to click. */
			width: 1.5em;
			display: inline-block;
			text-align: center;
		}

		.permalink:visited {
			color: rgb(0, 0, 238);
		}

		.permalink:hover {
			color: #66f;
		}

		h3 .permalink {
			font-size: 16px;
		}

		.endpoint {
			position: relative;
			background-color: #fff;
			border: 1px solid #b7b7b7;
			margin-bottom: -1px;
			padding: .2em .5em;
			border-radius: 2px;
		}

		.endpoint-top {
			cursor: pointer;
		}

		.endpoint-info {
			margin-left: 4.5rem;
			display: none;
		}

		.endpoint-info p {
			max-width: 55em;
		}

		.resource {
			display: inline-block;
			min-width: 38rem;
		}

		.resource .method {
			display: inline-block;
			min-width: 4rem;
		}

		.param-name {
			display: inline-block;
			min-width: 11rem;
		}
	</style>
</head>

<body>
	<h1>{{.Config.Title}} API documentation {{.Config.Version}}</h1>

	{{if .Config.Description}}<p>{{.Config.Description}}</p>{{end}}
	{{if .Config.ContactEmail}}
		<p>
			Contact <a href="mailto:{{.Config.ContactEmail}}">
				{{if .Config.ContactName}}
					{{.Config.ContactName}}
				{{else}}
					{{.Config.ContactEmail}}
				{{end}}
			</a> for questions.
		</p>
	{{end}}

	{{define "paramsTpl"}}
		<ul>
			{{range $p := .Params}}
				<li><code class="param-name">{{$p.Name}}</code> {{$p.Info}}</li>
			{{end}}
		</ul>
	{{end}}

	<h2>Endpoints</h2>
	{{range $i, $e := .Endpoints}}
		{{if eq $i 0}}
			</div><div>
			<h3 id="{{index $e.Tags 0}}" class="js-expand">
				{{index $e.Tags 0}}
				<a class="permalink" href="#{{index $e.Tags 0}}">§</a>
			</h3>
		{{else if ne (index (index $.Endpoints (add $i -1)).Tags 0) (index $e.Tags 0)}}
			</div><div>
			<h3 id="{{index $e.Tags 0}}" class="js-expand">
				{{index $e.Tags 0}}
				<a class="permalink" href="#{{index $e.Tags 0}}">§</a>
			</h3>
		{{end}}

		<div class="endpoint" id="{{$e.Method}}-{{$e.Path}}">
			<div class="endpoint-top">
				<code class="resource"><span class="method">{{$e.Method}}</span> {{$e.Path}}</code>
				{{$e.Tagline}}
				<a class="permalink" href="#{{$e.Method}}-{{$e.Path}}">§</a>
			</div>

			<div class="endpoint-info">
				<p>{{$e.Info}}</p>

				{{if $e.Request.Path}}
					<h4>Path parameters</h4>
					{{/* {{template "paramsTpl" $e.Request.Path}} */}}
				{{end}}

				{{if $e.Request.Query}}
					<h4>Query parameters</h4>
					{{/* {{template "paramsTpl" $e.Request.Query}} */}}
				{{end}}

				{{if $e.Request.Form}}
					<h4>Form parameters</h4>
					{{/* {{template "paramsTpl" $e.Request.Form}} */}}
				{{end}}

				{{if $e.Request.Body}}
					<h4>Request body</h4>
					<ul>
						<li><a href="#{{$e.Request.Body.Reference}}">{{$e.Request.Body.Reference}}</a>
							<sup>({{$e.Request.ContentType}})</sup></li>
					</ul>
				{{end}}

				<h4>Responses</h4>
				<ul>
					{{range $code, $r := $e.Responses}}
						<li><code class="param-name">{{$code}} {{status $code}}</code>
							{{if $r.Body}}
								{{if $r.Body.Reference}}
									<a href="#{{$r.Body.Reference}}">{{$r.Body.Reference}}</a>
								{{else}}
									{{$r.Body.Description}}
								{{end}}
								<sup>({{$r.ContentType}})</sup>
							{{end}}
						</li>
					{{end}}
				</ul>
			</div>
		</div>
	{{end}}

	<h2>Models</h2>
	{{range $k, $v := .References}}
		<h3 id="{{$k}}">
			{{$k}}
			<a class="permalink" href="#{{$k}}">§</a>
		</h3>
		<div class="endpoint">
			<p>{{$v.Info}}</p>
			<pre>{{$v.Schema|schema}}</pre>
		</div>
	{{end}}

	<script>
		var add = function(endpoint) {
			// Expand row on click.
			var topLine = endpoint.getElementsByClassName('endpoint-top')[0]
			var info = endpoint.getElementsByClassName('endpoint-info')[0]
			topLine.addEventListener('click', function(e) {
				if (e.target.className === 'permalink')
					return

				e.preventDefault()
				//for (var i = 0; i < topLine.length; i++)
				info.style.display = info.style.display === 'block' ? '' : 'block'
			})

			// Prevent text selection on double click.
			//endpoint.addEventListener('mousedown', function(e) {
			//	if (e.detail > 1)
			//		e.preventDefault()
			//})
		}

		var ep = document.getElementsByClassName('endpoint')
		for (var i = 0; i < ep.length; i++)
			add(ep[i])

		// Expand all rows in the section.
		document.addEventListener('click', function(e) {
			if (e.target.className !== 'js-expand')
				return

			e.preventDefault()
			var parent = e.target.parentNode
			if (parent.tagName.toLowerCase() === 'h3')
				parent = parent.parentNode

			var info = parent.getElementsByClassName('info')
			for (var i = 0; i < info.length; i++)
				info[i].style.display = info[i].style.display === 'block' ? '' : 'block'
		})
	</script>
</body>
</html>
`))

// WriteHTML writes w as HTML.
func WriteHTML(w io.Writer, prog *docparse.Program) error {
	// Too hard to write template otherwise.
	for i := range prog.Endpoints {
		prog.Endpoints[i].Path = prog.Config.Basepath + prog.Config.Prefix + prog.Endpoints[i].Path

		if len(prog.Endpoints[i].Tags) == 0 {
			prog.Endpoints[i].Tags = []string{"default"}
		}
	}

	return mainTpl.Execute(w, prog)
}

// ServeHTML serves HTML documentation at addr.
func ServeHTML(addr string) func(io.Writer, *docparse.Program) error {
	return func(_ io.Writer, prog *docparse.Program) error {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Rescan, but first clear some fields so we don't end up with
			// duplicate data.
			prog.Config.Output = func(io.Writer, *docparse.Program) error {
				return nil
			}
			prog.Endpoints = nil
			prog.References = make(map[string]docparse.Reference)

			err := docparse.FindComments(os.Stdout, prog)
			if err != nil {
				w.WriteHeader(500)
				_, wErr := fmt.Fprintf(w, "could not parse comments: %v", err)
				if wErr != nil {
					_, _ = fmt.Fprintf(os.Stderr, "could not write response: %v", wErr)
				}

				return
			}

			err = mainTpl.Execute(w, prog)
			if err != nil {
				_, wErr := fmt.Fprintf(w, "could not execute template: %v", err)
				if wErr != nil {
					_, _ = fmt.Fprintf(os.Stderr, "could not write response: %v", wErr)
				}
			}
		})

		fmt.Printf("serving on %v\n", addr)
		return http.ListenAndServe(addr, nil)
	}
}
