package package_alias

import alias "package-alias/otherpkg"

/*
   Testing here that import aliases work as expected, we can
   refer to it via the alias or its actual package name and it
   will end up in the response as otherpkg.Something rather than
   alias.Something
*/

// OtherThing is another thing.
type OtherThing struct {
	Something alias.Something
}

type ErrorResponse struct {
	Errors []alias.Error
}

// POST /foo/{id} foobar
//
// Request body (application/json): OtherThing
// Form: alias.Something
// Response 200: otherpkg.Something
// Response 201: alias.Something
// Response 400: ErrorResponse
