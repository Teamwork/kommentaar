package example

import "fmt"

// POST /foo/:ID foobar hello
// Create a new foo.
//
// This will create a new foo object for a customer. It's important to remember
// that only Pro customers have access to foos.
//
// Form:
//   id:      ID of the object (number, required)
//   subject: The subject (pattern: [a-z], example: "woot", "bar")
//
// Query:
//   same_format: (string, optional)
//   an_array:    (array[string])
//   woot:        just a desc
//   zxc:         just a desc (required,
//				               pattern: [a-z])
//   qwe:         (required)
//   hm:          How about a multi line description? How are we going to do
//                that? I think just by indentation?
//
// Path:
//   ID: The foo ID.
//
// Request body (application/json):
//   $object: RequestObj
//
// Response 200 (application/json):
//   $object: AnObject
//
// Response 400 (application/json):
//   $object: ErrorObject
//
// These docs are general Go docs, and not parsed (note the blank line).
// Actually, the above OpenAPI block could be anywhere in the code; and doesn't
// *have* to be right above the handler.
func MyHandler() {
	x := RequestObj{}
	y := AnObject{}
	fmt.Println(x, y)
}

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
	// Just any comment here really (number, required)
	ID int

	// Document it!
	Subject string

	// Document it!
	XSubject string
}

// ErrorObject ..
type ErrorObject struct {
	// Errors..
	Errors []string
}

func main() {
}
