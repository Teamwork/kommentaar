package embedded_sometimes

// RequestObj is a request object.
type RequestObj struct {
	CommonProperties // embedded here
}

// AnObject is another object.
type AnObject struct {
	Properties CommonProperties // named here
}

// CommonProperties has common properties.
type CommonProperties struct {
	Field string
}

// POST /foo/{id} foobar
//
// Request body (application/json): RequestObj
// Response 200 (application/json): AnObject
