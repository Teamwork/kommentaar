// Package html outputs to HTML.
package html

import (
	"html/template"
	"io"

	"github.com/teamwork/kommentaar/docparse"
)

var tpl = template.Must(template.New("out").Funcs(template.FuncMap{
	"add": func(a, b int) int { return a + b },
}).Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>{{.Config.Title}} API documentation {{.Config.Version}}</title>

	<style>
		body {
			font: 16px/1.9em sans-serif;
		}

		a {
			text-decoration: none;
		}

		p {
			margin: 0;
			padding: 0;
		}

		h2 {
			color: #3a6ea5;
		}

		h3 {
			margin: 0;
		}

		.endpoint {
			background-color: #f7f7f7;
			border: 1px solid #b7b7b7;
			margin-bottom: .5rem;
			padding: .2em .5em;
		}

		.info {
			display: none;
		}

		.resource {
			display: inline-block;
			width: 38rem;
		}

		.resource .method {
			display: inline-block;
			width: 3.7rem;
		}

		strong {
			display: block;
		}
		strong, p {
			margin-left: .5rem;
		}
	</style>
</head>

<body>
	<h1>{{.Config.Title}} API documentation {{.Config.Version}}</h1>

	{{range $i, $e := .Endpoints}}
		{{if eq $i 0}}
			<h2>{{index $e.Tags 0}}</h2>
		{{else if ne (index (index $.Endpoints (add $i -1)).Tags 0) (index $e.Tags 0)}}
			<h2>{{index $e.Tags 0}}</h2>
		{{end}}

		<div class="endpoint" id="{{$e.Method}}-{{$e.Path}}">
			<a href="#{{$e.Method}}-{{$e.Path}}">§</a>
			<a href="#" class="js-expand">⬇</a>
			<code class="resource"><span class="method">{{$e.Method}}</span> {{$e.Path}}</code>
			{{$e.Tagline}}

			<div class="info">
				<p>{{$e.Info}}</p>

				<h3>Request</h3>
				{{if $e.Request.Path}}
					<strong>Path parameters</strong>
					<p>
						{{range $p := $e.Request.Path.Params}}
							{{$p.Name}} {{$p.Info}}<br>
						{{end}}
					</p>
				{{end}}

				{{if $e.Request.Query}}
					<strong>Query parameters</strong>
					<p>
						{{range $p := $e.Request.Query.Params}}
							{{$p.Name}} {{$p.Info}}<br>
						{{end}}
					</p>
				{{end}}

				{{if $e.Request.Form}}
					<strong>Form parameters</strong>
					<p>
						{{range $p := $e.Request.Form.Params}}
							{{$p.Name}} {{$p.Info}}<br>
						{{end}}
					</p>
				{{end}}

				{{if $e.Request.Body}}
					<strong>Body</strong>
					<p>{{$e.Request.ContentType}} <a href="#TODO">{{$e.Request.Body.Reference}}</a></p>
				{{end}}

				<h3>Responses</h3>
				{{range $code, $r := $e.Responses}}
					<strong>{{$code}}</strong>
					<p>{{$r.ContentType}} {{if $r.Body}}<a href="#TODO">{{$r.Body.Reference}}</a>{{end}}</p>
				{{end}}
			</div>
		</div>
	{{end}}

	<script src="https://code.jquery.com/jquery-3.3.1.min.js"></script>
	<script>
	$('.js-expand').on('click', function(e) {
		e.preventDefault()
		info = $(this).parent().find('.info')
		info.css('display', info.is(':visible') ? '' : 'block')
	})
	/*
		document.getElementsByClassName('js-expand').addEventListener('click', function(e) {
			e.preventDefault()
			this.parentNode.getElementsByClassName('info')[0].style.display = 'block'
		})
		*/
	</script>
</body>
</html>
`))

// WriteHTML writes w as HTML.
func WriteHTML(w io.Writer, prog docparse.Program) error {
	return tpl.Execute(w, prog)
}
