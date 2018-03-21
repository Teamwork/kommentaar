[![Build Status](https://travis-ci.org/Teamwork/kommentaar.svg?branch=master)](https://travis-ci.org/Teamwork/kommentaar)
[![codecov](https://codecov.io/gh/Teamwork/kommentaar/branch/master/graph/badge.svg)](https://codecov.io/gh/Teamwork/kommentaar)

Generate documentation for Go APIs.

The primary focus is currently on [OpenAPI](https://github.com/OAI/OpenAPI-Specification)
output (previously known as Swagger), but it can also output directly to HTML,
and the design allows easy addition of other output formats.

Goals:

- Easy to use.
- Good performance.
- Will not require significant code refactors to use in most cases.

Non-goals:

- Support every single last OpenAPI feature.

Using the tool
--------------

    $ go install github.com/teamwork/kommentaar

Get all comments from one package:

    $ kommentaar github.com/teamwork/desk/api/v1/inboxController

Or from a package and all subpackages:

    $ kommentaar github.com/teamwork/desk/api/...

The default output is as an OpenAPI 3 YAML file, which is *not* compatible with
the OpenAPI 2/Swagger specification.

See `kommentaar -h` for a list of options. You can also the Go API (see godoc).

Syntax
------

The Kommentaar syntax is primarily driven by a simple data format in Go
comments. While "programming-by-comments" is not always ideal, using comments
does make it easier to use in some scenarios as it doesn't assume too much about
how you write your code.

A simple example:

```go
type createResponse struct {
    ID    int     `json:"id"`
    Price float64 `json:"price"`
}

// POST /bike bikes
// Create a new bike.
//
// This will create a new bike. It's important to remember that newly created
// bikes are *not* automatically fit with a steering wheel or seat, as the
// customer will have to choose one later on.
//
// Form:
//   size:  The bike frame size in centimetre {int, required}.
//   color: Bike color code {string}.
//
// Response 200: $ref: createResponse
func create(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

To break it down:

- The comment block is *not* directly tied to the handler function. You will
  usually want to use it like above, but there is nothing stopping you from
  putting it in another file or even package.

- The first line *must* be the HTTP method followed by the path. The method
  *must* be upper-case, and the path *must* start with a `/`.

  The path can optionally be followed by one or more space-separated tags, which
  are used to group endpoints.

- The second line is used as a "tagline". This can only be a single line and
  *must* immediately follow the opening line with no extra newlines. This line
  is optional but highly recommended to use.

- After a single blank line any further text will be treated as the endpoint's
  description. This is free-form text and may be omitted (especially in cases
  where it just repeats the tagline it's not useful to add).

- The `Form:` header denotes the form parameters, the parameter list *should* be
  indented by one or more spaces and is as `name: description {keywords}`. The
  keywords are optional.

- The response references a struct, `createResponse` in this case.

See the [`example/` directory](/example) for a more elaborate example, and
[`doc/syntax.markdown`](doc/syntax.markdown) for a full description of the
syntax.
