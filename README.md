[![Build Status](https://travis-ci.org/Teamwork/kommentaar.svg?branch=master)](https://travis-ci.org/Teamwork/kommentaar)
[![codecov](https://codecov.io/gh/Teamwork/kommentaar/branch/master/graph/badge.svg)](https://codecov.io/gh/Teamwork/kommentaar)

Generate OpenAPI files from comments in Go files.

The idea is that you can write documentation in your comments in a simple
readable manner.

Using the tool
==============

    $ go install github.com/teamwork/kommentaar

Get all comments from one package:

    $ kommentaar github.com/teamwork/desk/api/v1/inboxController

Or from a package and all subpackages:

    $ kommentaar github.com/teamwork/desk/api/...

The default output is as an OpenAPI 3 YAML file, which is *not* compatible with
the OpenAPI 2/Swagger spec.

Not everything is correctly output yet. It's a work-in progress tool.

Syntax
======

A simple example:

    // POST /foo foobar
    // Create a new foo.
    //
    // This will create a new foo object for a customer. It's important to remember
    // that only Pro customers have access to foos.
    //
    // Form:
    //   subject: The subject {string, required}.
    //   message: The message {string}.
    //
    // Response 200 (application/json):
    //   $object: responseObject

    type responseObject struct {
        // Unique identifier.
        ID int `json:"id"`

        // All threads that belong to this foo.
        Threads []models.Threads
    }

To break it down:

- The first line *must* be the HTTP method followed by the path. The method
  *must* be upper-case, and the path *must* start with a /.
  The path can optionally be followed by one or more space-separated tags, which
  can are used to group endpoints (you typically want the controller name here,
  e.g. `tickets` or `inboxes`).

- The second line is used as a "tagline". This can only be a single line and
  *must* immediately follow the opening line with no extra newlines. This is
  optional.

- After a single blank line any further text will be treated as the endpoint's
  description. This is free-form text, this may be omitted (especially in cases
  where it just repeats the tagline it's not very useful to add).

- The rest is is formatted as parameter lists which start with a header. A
  header is any text ending in a semicolon (`:`). The block contents is indented
  with two spaces as a matter of convention to aid readability, but this is not
  mandatory.

- The following headers can be used:

  - `Path:`, `Query:`, `Form:`  – Parameters for the path, query, and form.
  - `Request body [(Content-Type)]:` – The request body.
  - `Response [code] [(Content-Type)]` – Response bodies; may occur more than
    once for different status codes.

- The parameter list can be specified in two ways, either as a reference to a
  struct or "inline". You need to choose one per list, and can't mix the two.

- An inline parameter looks like `name: a description {tag1, tag2}`.

- A reference is denoted by `$object: pkg name`. The `pkg` may be omitted, in
  which case the package the file is in is used. Struct fields can be documented
  with the same tags as inline parameters.

See the [`example` directory](/example) for a more elaborate example.
