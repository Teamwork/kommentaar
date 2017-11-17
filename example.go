package main

import "fmt"

// POST /foo tickets
//
// Post a new foo object.
//
// Query:
//   same_format  (string, optional)
//   an_array     (array[string])
//   woot:        just a desc
//   zxc:         just a desc (required)
//   qwe          (required)
//
// Request body (form):
//   object: arp242.net/kommentaar.requestObj
//
// Response 200:
//    object: arp242.net/kommentaar.AnObject
//
// Response 400 (application/json):
//    object: arp242.net/kommentaar.AnObject

// The docs
func myHandler() {
	x := requestObj{}
	y := anObject{}
	fmt.Println(x, y)
}

// requestobj is now documented
type requestObj struct {
	// woot woot
	asd string

	// HelloasXXXX
	zxcvzxcv string

	// Hello qwer
	qqzxcvzxcv string

	// Hello asdf
	zxcvxzxcvzxcv string
}

type anObject struct {
	// Just any comment here really (number, required)
	ID int

	// Document it!
	Subject string

	// Document it!
	XSubject string
}
