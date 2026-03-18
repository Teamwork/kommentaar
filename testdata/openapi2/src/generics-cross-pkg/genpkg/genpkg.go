package genpkg

type CustomType string
type MyGeneric[T any] struct {
	// Value is the generic value.
	Value T
	// Count is a fixed field.
	Count int
}

type MyGenericMulti[T, N any] struct {
	// First is the first type param field.
	First T
	// Second is the second type param field.
	Second N
}
