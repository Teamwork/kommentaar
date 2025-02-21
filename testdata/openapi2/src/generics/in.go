package generics

type myGeneric[T, N any] struct {
	// This is a simple field.
	Field1 T
	// This is a array field.
	Field2 []T
	// This is a map field.
	Field3 map[string]T
	// This is another simple field.
	Field4 int
	// This is a different tag field.
	Field5 N `json:"hello5"`
	// This is a different tag field with pointer.
	Field6 *N `json:"hello6"`
}

type reqRef struct {
	// Foo documents a generic type.
	Foo myGeneric[string, float64]
}

// POST /path
//
// Request body: reqRef
// Response 200: {empty}
