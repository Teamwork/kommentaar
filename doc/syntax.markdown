This document is not yet complete!

Kommentaar syntax
=================

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
