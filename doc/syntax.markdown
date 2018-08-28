This document is not yet complete!

Kommentaar syntax
=================

ABNF description
----------------

Syntax as [ABNF](https://tools.ietf.org/html/rfc5234).

    path-description   = verb path [ tag ] LF
    verb               = "GET" / "HEAD" / "POST" / "PUT" / "PATCH" / "DELETE" / "CONNECT" / "OPTIONS" / "TRACE"
    path               = path-absolute  ; https://tools.ietf.org/html/rfc3986#section-3.3
    tag                = *(ALPHA / DIGIT)

    param-alpha        = ; any Unicode character except "{", "}", ",", " "
    param-property     = ("{" param-alpha [":" param-alpha [param-alpha] ] *("," param-property) "}"

Description
-----------

Kommentaar is driven by *comment blocks*, which can appear as either multi-line
comments (`/* .. */`) or a block of single-line comments (`// ...`).

While "programming-by-comments" is not always ideal, it can be easier to use
than getting all information from the Go code as it doesn't assume too much
about how you write your code.

It's customary to put the comment block somewhere near the handler being
documented, but it may appear anywhere – even in a different package.

The general structure looks like:

    // POST /foo/{id} tag
    // One-line description.
    //
    // A more detailed multi-line description.
    //
    // Query: $ref: QueryObj
    // Request body: $ref: RequestObj
    // Response 200: $ref: AnObject
    // Response 400: $ref: ErrorObject

Path description, tagline, and description
------------------------------------------
Every Kommentaar comment block must start with a description of the path as:

    VERB /path [tag]

- The `VERB` is a valid HTTP verb; it *must* be in upper-case.
- The `/path` is a HTTP path. Path parameters can be added as `{..}`.
- An optional tag can be added for categorisation.

You can use multiple path descriptions:

    VERB /path [tag]
    OTHER /anotherpath [tag]

The line immediately following the path descriptions is used as a "tagline".
This can only be a single line and *must* immediately follow the path
description with no extra newlines. This line is optional but highly recommended
to use.

The tagline can be of any length, but it is highly recommended that it is kept
short and concise.

After a single blank line any further text will be treated as the endpoint's
description. This is free-form text and may be omitted (especially in cases
where it just repeats the tagline it's not useful to add).

Full example:

    POST /bike/{shedID} bikes
    Create a new bike and store it in the given shed.

    It's important to remember that newly created bikes are *not* automatically
    fit with a steering wheel or seat, as the customer will have to choose one
    later on.

Path, query, and form parameters
--------------------------------

    Form: $ref: formParams
    Path: $ref: formParams
    Query: $ref: formParams

Request body
------------

    Request body: $ref: createRequest
    Request body (application/json): $ref: createRequest

Responses
---------

    Response: $ref: createResponse
    Response 200: $ref: createResponse
    Response 204: $empty
    Response 400: $default
    Response 404 (application/json): $default

Parameter properties
--------------------

Comments documenting struct fields can have *parameter properties* with special
keywords to document them. These must appear within `{` and `}` characters, and
may appear multiple times. A single `{..}` block may also contain multiple
properties separated by a `,`.

These values are removed from the documentation string and will be added as
special fields in the output format.

Supported parameters:

- `required`        – parameter must be given.
- `optional`        – parameter can be blank; this is the default, but
                      specifying it explicitly may be useful in some cases.
- `default: v1`     – default value.
- `enum: v1 v2 ...` – parameter must be one one of the values.
- `range: n-n`      – parameter must be within this range; either number can be
                      `0` to indicate there is no lower or upper limit (only
                      useful for numeric parameters).
- Any [format from JSON schema](https://tools.ietf.org/html/draft-handrews-json-schema-validation-01#section-7.3)

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
