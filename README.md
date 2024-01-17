[![GoDoc](https://godoc.org/github.com/teamwork/kommentaar?status.svg)](https://godoc.org/github.com/teamwork/kommentaar)
[![Build Status](https://travis-ci.com/Teamwork/kommentaar.svg?branch=master)](https://travis-ci.com/Teamwork/kommentaar)
[![codecov](https://codecov.io/gh/Teamwork/kommentaar/branch/master/graph/badge.svg)](https://codecov.io/gh/Teamwork/kommentaar)

Kommentaar generates documentation for Go APIs.

The primary focus is currently on [OpenAPI](https://github.com/OAI/OpenAPI-Specification)
output (previously known as Swagger), but it can also output directly to HTML,
and the design allows easy addition of other output formats.

Goals:

- Easy to use.
- Good performance.
- Will not require significant code refactors to use in almost all cases.
- Impossible to produce invalid OpenAPI files; prefer being strict over
  flexible.

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
example to serve documentation in an HTTP endpoint.

### Usage with Docker


There is a kommentaar Docker image available at `ghcr.io/teamwork/kommentaar` which in some cases may simplify the setup process.
It takes in the following parameters:

- `MODULE_PATH` env var (required) - path to your module, e.g. `github.com/teamwork/amazingapp`
- `EXEC_PATH` env var (optional) - path passed to kommentaar - can be relative or absolute, e.g. `./api/v3/...`, or `/go/src/github.com/teamwork/amazingapp/api/v3/...`. Default value is `/go/src/$MODULE_PATH/...`
- `CONFIG_NAME` env var (optional) - name of kommentaar config file in the `/config` volume (see below). Default value is `kommentaar.conf`
- `/code` volume (required) - folder containing your go code
- `/config` volume (optional) - folder containing your kommentaar config file, by default the included `config.example` file is copied there as `kommentaar.conf`
- `/output` volume (required) - folder where the generated `swagger.yaml` file will be placed.

Usage examples:
- `docker run --rm -it -e MODULE_PATH="github.com/teamwork/amazingapp" -v .:/code -v .:/config -v .:/output ghcr.io/teamwork/kommentaar`
  - This assumes you are running the command in `amazingapp` folder
  - This is roughly equivalent to `kommentaar -config ./kommentaar.conf github.com/teamwork/amazingapp... > ./swagger.yaml`
- `docker run --rm -it -e MODULE_PATH="github.com/teamwork/amazingapp" -e EXEC_PATH="./api/v3/controller/..." -e CONFIG_NAME=".kommentaar.v3" -v $HOME/dev/amazingapp:/code -v $HOME/dev/amazingapp/config:/config -v %HOME/dev/amazingapp/api/v3:/output ghcr.io/teamwork/kommentaar`
  - This will run the equivalent of `kommentaar -config $HOME/dev/amazingapp/config/.kommentaar.v3 $GOPATH/src/github.com/teamwork/amazingapp/api/v3/controller/... > $HOME/dev/amazingapp/api/v3/swagger.yaml`


Syntax
------

See [`doc/syntax.markdown`](doc/syntax.markdown) for a full description of the
syntax; a basic example:

```go
type bikeRequest struct {
	// Frame colour {enum: black red blue, default: black}.
	Color string

	// Frame size in centimetres {required, range: 40-62}.
	Size int
}

type bikeResponse struct {
	// Price in Eurocents.
	Price int

	// Estimated delivery date {date}.
	DeliveryTime int
}

type errorResponse struct {
	Error []string `json:"errors"`
}

// POST /bike/{id} bikes
// Order a new bike.
//
// A more detailed multi-line description.
//
// Request body: bikeRequest
// Response 200: bikeResponse
// Response 400: errorResonse
```

Configuration
-------------

Kommentaar can be configured with a configuration file; see
[`config.example`](config.example) for the documentation.

Running Tests
-------------

`GO111MODULE=off ./bin/test ./...`

Motivation and rationale
------------------------

The motivation for writing Kommentaar was a lack of satisfaction with existing
tools:

- [yvasiyarov/swagger](https://github.com/yvasiyarov/swagger) requires extensive
  comments; you will need to duplicate every parameter as `@param foo query
  string Some description`. It's flexible but also tedious and ugly.

- We implemented [go-swagger](https://github.com/go-swagger/go-swagger) but
  found several pain points:

  - Implementing it meant doing a significant rewrite of our code base and a lot
	of "glue code", which made the code uglier in the opinion of many
	developers.
  - Slow; about 30 seconds for a fairly limited amount of endpoints (Kommentaar
	takes about 1.5 seconds for the same).
  - Very easy to generate invalid OpenAPI files.
  - Complex codebase made it hard to figure out why it was doing what it was
	doing.

- [goa](https://github.com/goadesign/goa) means a complete rewrite of our API,
  and whether the goa DSL approach is a good one is also debatable (we haven't
  tried it due to the prohibitive costs of the rewrite, so lack direct
  experience).

We tried implementing both yvasiyarov/swagger and go-swagger, and both ended in
fairly dismal (and time-consuming) failure.

Kommentaar is designed to strike a reasonable balance:

- You will need to duplicate *some* information from the code in comments, but
  not too much, and it shouldn't have to be updated very often; adding new
  request or response parameters is still easy.

- Makes *some* assumptions about your code (e.g. that you're returning a
  `struct`), but not many, and rewriting existing code (e.g. handlers returning
  a `map[string]interface{}`) should be straightforward.

- Syntax is straightforward and easy to read and write.

- Impossible to make Kommentaar output invalid OpenAPI files (if it does, then
  that's a bug); the syntax doesn't offer too much flexibility, and the tool
  errors out when it encounters unexpected or wrong input.

- Reasonably fast and should not exceed more than 2 or 3 seconds for
  moderate-sized APIs (and it can probably be made faster with some effort).
