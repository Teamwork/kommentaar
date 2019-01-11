Kommentaar syntax
=================

Introduction
------------

Kommentaar is a program to document Go APIs driven by a special syntax inside
comment blocks. This document describes the syntax as recognized by Kommentaar.

While "programming-by-comments" is not always ideal, it can be easier to use
than getting all information from the Go code, as it doesn't assume too much
about how the code is written. Also see the [Motivation and
rationale][rationale] section in the README file.

Augmented Backus-Naur Form (ABNF) is used as described in [RFC 5234][rfc5234].

Quick overview
--------------

Kommentaar is driven by a special syntax inside comment blocks, which can appear
as either multi-line comments (`/* .. */`) or a continues block of single-line
comments (`// ...`).

It's customary to put the comment block somewhere near the handler or route
definition being documented, but it may appear anywhere – including a different
package.

The general structure looks like:

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

This describes the endpoint `POST /bike/{id}`, which will be grouped under
`bikes`. It describes the JSON request body in the `bikeRequest` struct and the
different responses for the various HTTP codes in `bikeResponse` and
`errorResonse`.

The request and response structs have various properties for validation
(`{required, range: 40-62}`) and formatting (`{date}`).

Path description, tagline, and description
------------------------------------------

Every Kommentaar comment block must start with a description of the path as
follows (and will be ignored if it doesn't):

    VERB /path [tag] [tag2...]

- The `VERB` is a valid HTTP verb; it must be in upper-case.
- The `/path` is a HTTP path. Path parameters can be added as `{..}`.
- One or more optional tags can be added for categorisation.

Multiple path descriptions are allowed:

    VERB /path [tag]
    OTHER /anotherpath [tag]

The line immediately following the path descriptions is used as a "tagline".
This can only be a single line and must immediately follow the path description
with no extra newlines. This line is optional but highly recommended to use. The
tagline can be of any length, but it is highly recommended that it is kept short
and concise.

After a single blank line any further text will be treated as the endpoint's
description. This is free-form text and may contain blank lines. It may be
omitted – especially in cases where it just repeats the tagline it's not useful
to add.

Inside the description section the `$print` operator is also supported to allow
referencing a constant or variable in the codebase to be added to the
documentation. This can allow you for example to avoid duplicating some common
text or output the value of a whitelist without duplicating it.

The description will end once the first reference directive is found. The
description cannot continue after reference directives.

    path-description   = verb path [ tag *( " " tag ) ] LF
    verb               = "GET" / "HEAD" / "POST" / "PUT" / "PATCH" / "DELETE" / "CONNECT" / "OPTIONS" / "TRACE"
    path               = path-absolute  ; https://tools.ietf.org/html/rfc3986#section-3.3
    tag                = *(ALPHA / DIGIT)

Full example:

    POST /bike/{id} bikes
    Order a new bike.

    It's important to remember that newly created bikes are *not* automatically
    fit with a steering wheel or seat, as the customer will have to choose one
    later on.

    Adding a steering wheel or seat can be done in the PATCH request.

    You can side-load addition data to save an extra request, the following
    relationships are available for side-loading with the include query
    paramter:
    $print postBikeIncludes

Reference directives
--------------------

The general structure for referencing types is:

    <keyword>: <ref>

The various values for `<keyword>` are described below.

`<ref>` refers to a type and can be in three formats:

- `t`                 – current package.
- `pkg.t`             – imported package (e.g. `import "import/path/pkg"`).
- `import/path/foo.t` – full import path; when the package isn't imported.

Only `struct` and `interface` types can be referenced. Interfaces don't have
fields and only the documentation for the interface will be added to the output.

Embedded structs are merged in to the parent struct, unless they have the
applicable struct tag (as configured with `struct-tag`), in which case they're
added as reference in the output.

References are looked up in the customary locations (vendor, GOPATH). Invalid
references are an error.

    ; https://golang.org/ref/spec#identifier
    ; https://golang.org/ref/spec#Qualified_identifiers
    ; https://golang.org/ref/spec#Import_declarations
    ref            = identifier / QualifiedIdent / ( ImportPath "." identifier )

Unexported fields are ignored; unexported fields with an applicable struct tag
are considered an error.

### Path, Query, and Form references

A `Path` reference can be used to document path parameters; for example:

    type pathParams {
        // Bike ID from the manufacturer.
        ID int `path:"id"`
    }

    // GET /bike/{id}
    //
    // Path: pathParams

Attempting to document path parameters that don't exist as `{..}` placeholders
in the path is an error.

The OpenAPI specification states that all parameters inside the path must have a
corresponding path parameter. Missing path parameters will be automatically
added in the (OpenAPI) output since explicitly documenting these is often
useless.

A `Query` reference can be used to document parameters from URLs; a `Form`
reference can be used to document form data (`application/x-www-form-urlencoded`
or `multipart/form-data`).

The `Form` and `Query` parameters follow the same format:

    Form: formParams
    Query: queryParams

Referencing Form, Path, or Query parameters will always use the `form`, `path`,
and `query` struct tags. A value of `-` means it will be ignored; no struct tag
means it will add the field name as-is.

    param-ref      = ( "Form" / "Path" / "Query" ) ": " ref LF

### Request body

The request body is any request body that is not a form; for example JSON, XML,
YAML, or something else.

    Request body: createRequest
    Request body (application/xml): createRequest

The first form will use the configured default Content-Type; the second form
explicitly defines it for this request body.

    content-type   = type-name "/" subtype-name  ; https://tools.ietf.org/html/rfc6838#section-4.2
    request-ref    = "Request body" [ "(" content-type ")" ] ": " ref LF

### Responses

Response bodies are mapped to a HTTP status code:

    Response 200: createResponse

Every endpoint must have at least one defined response.

The response code `200` will be used it it's omitted:

    Response: createResponse

You can define a Content-Type like with Request bodies; it will use the
configured default Content-Type if omitted:

    Response 404 (application/json): createResponse

The `{empty}` keyword indicates that this response code may be returned, but
without any code. In general this should only be used for `204 No Content`:

    Response 204: {empty}

The `{data}` keyword indicates that this response returns unstructured data,
such as text, HTML, an image, spreadsheet document, etc. An explicit
Content-Type is required when using `{data}`.

    Response 200 (text/plain): {data}

The `{default}` keyword indicates it should use the default reference for this
response code:

    Response 400: {default}

It is an error if no default reference is configured for this response code.

    response-ref   = "Response" [ 3DIGIT ] ":" [ "(" content-type ")" ] ( "{empty}" / "{default}" / ": " ref ) LF

References
----------

### Parameter properties

Comments documenting struct fields can have *parameter properties* with special
keywords to document them. These must appear within `{` and `}` characters and
may appear multiple times. A single `{..}` block may also contain multiple
properties separated by a `,`.

These values are removed from the documentation string and will be added as
special fields in the output format.

Supported parameters:

- `omitdoc`         – parameter is not added to the generated output.
- `required`        – parameter must be given.
- `optional`        – parameter can be blank; this is the default, but
                      specifying it explicitly may be useful in some cases.
- `readonly`        – parameter cannot be set by the user from the request body
                      or query/form parameters. Attempting to set it will be or
                      result in an error.
- `default: v1`     – default value.
- `enum: v1 v2 ..`  – parameter must be one one of the values.
- `range: n-n`      – parameter must be within this range; either number can be
                      `0` to indicate there is no lower or upper limit (only
                      useful for numeric parameters).
- `schema: path`    – use a JSON schema file (as JSON as YAML) to describe this
                      parameter, ignoring the Kommentaar directives for it. The
                      path is relative to the file in which it's found.
- Any [format from JSON schema][json-schema-format].

Examples:

    type paginate struct {
        // Page to retrieve {required}.
        Page int

        // Number of results in a single page {range: 20-100, default: 20}.
        PageSize int

        // Sorting order {enum: asc desc} {default: asc}.
        Order int

        // Only fetch results updated since this time {date-time}.
        UpdatedSince time.Time
    }

Using unknown keywords is an error.

    param-alpha    = ; any Unicode character except "{", "}", ",", " "
    param-property = "{" param-alpha [ ":" param-alpha [ param-alpha ] ] *( "," param-property ) "}"


[rationale]: https://github.com/Teamwork/kommentaar#motivation-and-rationale
[rfc2119]: https://tools.ietf.org/html/rfc2119
[rfc5234]: https://tools.ietf.org/html/rfc5234
[json-schema-format]: https://tools.ietf.org/html/draft-handrews-json-schema-validation-01#section-7.3
