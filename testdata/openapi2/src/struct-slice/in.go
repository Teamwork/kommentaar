package path

type resp struct {
	Basic     []string          `json:"basic"`     // Basic comment.
	Custom    mySlice           `json:"custom"`    // Custom comment.
	Double    anotherSlice      `json:"another"`   // Double comment.
	StructRef customFieldValues `json:"structRef"` // structRefComment.
	deal      deal
}

// mySlice comment.
type mySlice []string

// anotherSlice comment.
type anotherSlice mySlice

type customFieldValues []*customFieldValue

type customFieldValue struct {
	val string
}

type deal struct {
	customFieldValues []*customFieldValue
}

// POST /path
//
// Response 200: $ref: resp
