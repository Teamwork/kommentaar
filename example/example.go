// nolint
package example

import (
	"fmt"

	"github.com/teamwork/kommentaar/example/exampleimport"
)

// POST /foo/{id} foobar
// Create a new foo.
//
// This will create a new foo object for a customer. It's important to remember
// that only Pro customers have access to foos.
//
// Request body (application/json): $ref: RequestObj
// Response 200 (application/json): $ref: AnObject
// Response 400 (application/json): $ref: ErrorObject
// Response 401 (application/json): $ref: exampleimport.Foo

// These docs are general Go docs, and not parsed (note the blank line).
// Actually, the above OpenAPI block could be anywhere in the code; and doesn't
// *have* to be right above the handler.
func MyHandler() {
	x := RequestObj{}
	y := AnObject{}
	_ = exampleimport.Foo{}
	fmt.Println(x, y)
}

type formParams struct {
	ID int64
}

// POST /second/endpoint
//
// Just to see if that works correct.
//
// Response 200: $empty

// Other others a lot!
func Other() {
}

// RequestObj is now documented.
type RequestObj struct {
	// woot woot
	Asd string

	// HelloasXXXX
	zxcvzxcv string

	// Hello qwer
	qqzxcvzxcv string

	// Hello asdf
	zxcvxzxcvzxcv string
}

type AnObject struct {
	// Just any comment here really (int, required)
	ID int

	// Document it!
	Subject string

	// Document it!
	XSubject string
}

// ErrorObject ..
type ErrorObject struct {
	// Errors..
	Errors []MyError
}

// MyError ..
type MyError struct {
	Message string
	Code    int
}

func main() {
}
