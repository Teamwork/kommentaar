Generate OpenAPI files from comments in Go files.

The idea is that you can write documentation in your comments in a simple
readable manner.

The first line must be method followed by the path, and optionally followed by
one or more space-separated tags. The method *must* be upper-case, and the path
*must* start with a /. This comment doesn't need to be the godoc comment for the
handler function.

	// POST /foo tickets

You can optionally provide a description; the first sentence will be used as the
short summary. The rest as the more detailed summary.

	// Post a new foo object.
	// This is where we explain some caveats or whatnot.

A form is request data sent as a form. The format for all the parameters is the
same, and all of these blocks are optional.

	// Form:
	//   id (number, required): ID of the object
	//   subject: The subject

URL query strings.

	// Query:
	//   same_format (string, optional)
	//   an_array (array[string])
	//   woot: just a desc

A request body in the given content type. The content-type specification is
optional; if it's not given it will default to the global value.

	// Request body (application/json):
	//   same_format (string, optional)

The response body for different status codes. The `200` code is actually
optional; it will be added automatically if it's missing. As with the Request
Body the content-type is optional.

	// JSON response 200 (application/json):

Reference an object; we will try and find the type and parse the comments from
that.

	//    object: arp242.net/kommentaar.anObject

A different response for a different status code.

	// JSON response 403 (application/jso):
	//   status: error
	//   message: A human-readable error message.

The object we referenced earlier.

	type anObject struct {

Every parameter has a comment which doesn't need to be in any specific format;
only the type and required are parsed, as with request parameters.

		// Just any comment here really (number, required)
		ID int

		// Document it!
		Subject string
	}

---

Putting the above all together, you get:

	// POST /foo tickets
	// Post a new foo object.
	//
	// This is where we explain some caveats or whatnot.
	//
	// Form:
	//   id (number, required): ID of the object
	//   subject: The subject
	//
	// Query:
	//   same_format (string, optional)
	//   an_array (array[string])
	//   woot: just a desc
	//
	// Request body (application/json):
	//   same_format (string, optional)
	//
	// JSON response 200 (application/json):
	//    object: arp242.net/kommentaar.anObject
	//
	// JSON response 403 (application/json):
	//   status: error
	//   message: A human-readable error message.
	func myHandler(r *http.Request, w *http.ResponseWriter) {
		// ...
	}

	type anObject struct {
		// Just any comment here really (number, required)
		ID int

		// Document it!
		Subject string
	}

Also see `example.go`.
