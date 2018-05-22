package path

type resp struct {
	Basic  []string     `json:"basic"`   // Basic comment.
	Custom mySlice      `json:"custom"`  // Custom comment.
	Double anotherSlice `json:"another"` // Double comment.
}

// mySlice comment.
type mySlice []string

// anotherSlice comment.
type anotherSlice mySlice

// POST /path
//
// Response 200: $ref: resp
