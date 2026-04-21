package generics_map_types

// Wrapper wraps a value with metadata.
type Wrapper[T any] struct {
	// Value is the wrapped value.
	Value T
	// Extra is additional metadata.
	Extra string
}

// GroupBy groups results by a field.
type GroupBy[T any] struct {
	// Field is the grouping field.
	Field T
	// Label is the display label.
	Label string
}

type reqRef struct {
	// FullKey is mapped via full instantiated key Wrapper[string].
	FullKey Wrapper[string]
	// MainKey is mapped via main type key GroupBy (matches all instantiations).
	MainKey GroupBy[int]
	// Unmapped is not in map-types so it expands as an object.
	Unmapped Wrapper[int]
}

// POST /path
//
// Request body: reqRef
// Response 200: {empty}
