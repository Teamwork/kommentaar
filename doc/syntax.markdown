This document is not yet complete!

Kommentaar syntax
=================

Description
-----------

Kommentaar is driven by *comment blocks*, which can appear as either multi-line
comments (`/* .. */`) or a block of single-line comments (`// ...`).

While "programming-by-comments" is not always ideal, using comments does make it
easier to use in some scenarios as it doesn't assume too much about how you
write your code.

Although it's customary to put the comment block somewhere near the handler
being documented, it may appear anywhere – even in a different package.

The general structure looks like:

	// POST /foo/{id} tag
	// One-line description.
	//
	// A more detailed multi-line description.
	//
	// Request body: $ref: RequestObj
	// Response 200: $ref: AnObject
	// Response 400: $ref: ErrorObject


Opening line, tagline, and description
--------------------------------------

    POST /bike bikes
    Create a new bike.

    This will create a new bike. It's important to remember that newly created
    bikes are *not* automatically fit with a steering wheel or seat, as the
    customer will have to choose one later on.

Path, query, and form parameters
--------------------------------

    Form:
      size:  The bike frame size in centimetre {int, required}.
      color: Bike color code {string}.

Request body
------------

    Request body: $ref: createRequest

Responses
---------

    Response 200: $ref: createResponse

    `$empty`

Parameter keywords
------------------

- `required`, `optional` –
- `string`, `integer`, `number`, `boolean` –
