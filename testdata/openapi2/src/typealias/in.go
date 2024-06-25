package typealias

// ReqRef is a request reference.
type ReqRef struct {
	Anonymous struct {
		Alias AliasExample `json:"alias,omitempty"`
	} `json:"anonymous,omitempty"`
}

// AliasExample is an example of type alias.
type AliasExample map[string]ExampleItem

// ExampleItem is an example item.
type ExampleItem struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

// POST /path
//
// Request body: ReqRef
// Response 200: {empty}
