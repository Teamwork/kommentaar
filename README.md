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

- Support every single last OpenAPI feature. Some features, such as different
  return values with `anyOf`, don't map well to how Go works, and supporting it
  would add much complexity and would benefit only a few users.

Using the tool
--------------

Install it:

    $ go get github.com/teamwork/kommentaar

Parse one package:

    $ kommentaar github.com/teamwork/desk/api/v1/inboxController

Or several packages subpackages:

    $ kommentaar github.com/teamwork/desk/api/...

The default output is as an OpenAPI 3 YAML file, which is *not* compatible with
the OpenAPI 2/Swagger specification.

See `kommentaar -h` for a list of options. You can also the Go API (see godoc).

Syntax
------

See [`doc/syntax.markdown`](doc/syntax.markdown) for a full description of the
syntax.
