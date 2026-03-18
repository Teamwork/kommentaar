package genpkg

// Wrapper is a generic struct defined in a separate package.
type Wrapper[T any] struct {
	// Value holds the generic field.
	Value T
	// Count is a fixed field.
	Count int
}
