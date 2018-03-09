package example

// DELETE /another/file
//
// ANOTHER FILE!
//
// Request body: $ref: net/mail.Address
// Response 200: $ref: AnObject
// Response 400: $ref: ErrorObject

// Response 401: $object: exampleimport.Foo
