package path

type resp struct {
	Basic     []string          `json:"basic"`     // Basic comment.
	Custom    mySlice           `json:"custom"`    // Custom comment.
	Double    anotherSlice      `json:"another"`   // Double comment.
	OneMore   oneMoreSlice      `json:"oneMore"`   // OneMore comment.
	StructRef customFieldValues `json:"structRef"` // structRefComment.
	Deal      deal              `json:"deal"`

	Nested  [][]item          `json:"nested"`
	MapRef  []map[string]item `json:"mapRef"`
	Iface   []interface{}     `json:"iface"`
	Generic []box[item]       `json:"generic"`
	Anon    []struct {
		Val string `json:"val"`
	} `json:"anon"`
}

type item struct {
	Name string `json:"name"`
}

type box[T any] struct {
	V T `json:"v"`
}

// mySlice comment.
type mySlice []string

// anotherSlice comment.
type anotherSlice mySlice

// oneMoreSlice comment.
type oneMoreSlice []map[string]any

type customFieldValues []*customFieldValue

type customFieldValue struct {
	Val string `json:"val"`
}

type deal struct {
	CustomFieldValues []*customFieldValue
}

// POST /path
//
// Response 200: resp
