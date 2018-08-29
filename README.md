[![GoDoc](https://godoc.org/github.com/teamwork/kommentaar?status.svg)](https://godoc.org/github.com/teamwork/kommentaar)
[![Build Status](https://travis-ci.org/Teamwork/kommentaar.svg?branch=master)](https://travis-ci.org/Teamwork/kommentaar)
[![codecov](https://codecov.io/gh/Teamwork/kommentaar/branch/master/graph/badge.svg)](https://codecov.io/gh/Teamwork/kommentaar)

Generate documentation for Go APIs.

The primary focus is currently on [OpenAPI](https://github.com/OAI/OpenAPI-Specification)
output (previously known as Swagger), but it can also output directly to HTML,
and the design allows easy addition of other output formats.

Goals:

- Easy to use.
- Good performance.
- Will not require significant code refactors to use in almost all cases.

Non-goals:

- Support every single last OpenAPI feature. Some features, such as different
  return values with `anyOf`, don't map well to how Go works, and supporting it
  would add much complexity and would benefit only a few users.

Using the tool
--------------

Install it:

    $ go get github.com/teamwork/kommentaar

Parse one package:

    $ kommentaar github.com/teamwork/desk/api/v1/inboxController

Or several packages:

    $ kommentaar github.com/teamwork/desk/api/...

The default output is as an OpenAPI 2 YAML file. You can generate a HTML page
with `-output html`, or directly serve it with `-output html -serve :8080`. When
serving the documentation it will rescan the source tree on every page load,
making development/proofreading easier.

See `kommentaar -h` for the full list of options.

You can also the [Go API](https://godoc.org/github.com/teamwork/kommentaar), for
example to serve documentation in a HTTP endpoint.

Syntax
------

See [`doc/syntax.markdown`](doc/syntax.markdown) for a full description of the
syntax.

Configuration
-------------

Kommentaar can be configured with a configuration file; see
[`config.example`](config.example) for the documentation.
