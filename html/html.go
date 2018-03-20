// Package html outputs to HTML.
package html

import (
	"html/template"
	"io"
	"net/http"

	"github.com/kr/pretty"
	"github.com/teamwork/kommentaar/docparse"
)

var funcMap = template.FuncMap{
	"add":    func(a, b int) int { return a + b },
	"status": func(c int) string { return http.StatusText(c) },
	"dump":   func(x interface{}) string { return pretty.Sprintf("%# v", x) },
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
			color: #fff;
			background-color: #888;
			border: 1px solid #888;
			margin-bottom: -1px;
		}

		h4 {
			margin: 0;
			font-size: 16px;
		}

		sup {
			color: #aaa;
		}

		.btn {
			color: #000;
			background-color: #ddd;
			border: 1px solid #aaa;
			border-radius: 1px;
			width: .8em;
			display: inline-block;
			text-align: center;
			line-height: 1em;
			padding: .2em;
		}

		.btn:visited {
			color: #000;
		}

		.btn:hover {
			background-color: #777;
		}

		.btn:first-child {
			margin-right: -1px;
		}

		h3 .btn {
			font-size: 16px;
			position: relative;
			top: -3px;
		}

		.btn-group {
			position: absolute;
			left: -1px;
			top: -4px;
			display: none;
		}

		h3:hover .btn-group,
		.endpoint:hover .btn-group {
			display: inline;
		}

		.endpoint {
			position: relative;
			background-color: #fff;
			border: 1px solid #b7b7b7;
			margin-bottom: .5rem;
			padding: .2em .5em;
			border-radius: 2px;
		}

		.info {
			margin-left: 4.5rem;
			display: none;
		}

		.resource {
			display: inline-block;
			min-width: 38rem;
		}

		.resource .method {
			display: inline-block;
			min-width: 3.7rem;
		}

		.param-name {
			display: inline-block;
			min-width: 11rem;
		}
	</style>
</head>

<body>
	<h1>{{.Config.Title}} API documentation {{.Config.Version}}</h1>

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
			<h3 id="{{index $e.Tags 0}}">
				{{index $e.Tags 0}}
				<span class="btn-group">
					<a class="btn" href="#{{index $e.Tags 0}}">§</a><a class="btn js-expand" href="#">⬇</a>
				</span>
			</h3>
		{{else if ne (index (index $.Endpoints (add $i -1)).Tags 0) (index $e.Tags 0)}}
			</div><div>
			<h3 id="{{index $e.Tags 0}}">
				{{index $e.Tags 0}}
				<span class="btn-group">
					<a class="btn" href="#{{index $e.Tags 0}}">§</a><a class="btn js-expand" href="#">⬇</a>
				</span>
			</h3>
		{{end}}

		<div class="endpoint" id="{{$e.Method}}-{{$e.Path}}">
			<code class="resource"><span class="method">{{$e.Method}}</span> {{$e.Path}}</code>
			{{$e.Tagline}}
			<span class="btn-group">
				<a class="btn" href="#{{$e.Method}}-{{$e.Path}}">§</a><a class="btn js-expand" href="#">⬇</a>
			</span>

			<div class="info">
				<p>{{$e.Info}}</p>

				{{if $e.Request.Path}}
					<h4>Path parameters</h4>
					{{template "paramsTpl" $e.Request.Path}}
				{{end}}

				{{if $e.Request.Query}}
					<h4>Query parameters</h4>
					{{template "paramsTpl" $e.Request.Query}}
				{{end}}

				{{if $e.Request.Form}}
					<h4>Form parameters</h4>
					{{template "paramsTpl" $e.Request.Form}}
				{{end}}

				{{if $e.Request.Body}}
					<h4>Request body</h4>
					<ul>
						<li><a href="#TODO">{{$e.Request.Body.Reference}}</a>
							<sup>({{$e.Request.ContentType}})</sup></li>
					</ul>
				{{end}}

				<h4>Responses</h4>
				<ul>
					{{range $code, $r := $e.Responses}}
						<li><code class="param-name">{{$code}} {{status $code}}</code>
							{{if $r.Body}}
								{{if $r.Body.Reference}}
									<a href="#TODO">{{$r.Body.Reference}}</a>
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
		<h3>{{$k}}</h3>
		<div class="endpoint">
			<p>{{$v.Info}}</p>
			<ul>
				{{range $i, $p := $v.Params}}
					<li>
						<code>{{$p.Name}}</code>
						<code>{{$p.Kind}}</code>
						– {{$p.Info}}
					</li>
				{{end}}
			</ul>
		</div>
	{{end}}

	<script>
		document.addEventListener('click', function(e) {
			window.eee = e
			if (e.target.className !== 'btn js-expand')
				return

			e.preventDefault()
			var parent = e.target.parentNode.parentNode
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
func WriteHTML(w io.Writer, prog docparse.Program) error {
	return mainTpl.Execute(w, prog)
}
