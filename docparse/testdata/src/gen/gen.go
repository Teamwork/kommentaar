package gen

import "net/mail"

type item struct {
	Name string `json:"name"`
}

type box[T any] struct {
	Data  T   `json:"data"`
	Items []T `json:"items"`
	Count int `json:"count"`
}

type pair[K, V any] struct {
	Key   K `json:"key"`
	Value V `json:"value"`
}

type list[T any] []T

type plain struct {
	Name string `json:"name"`
}

type generic struct {
	local    box[item]
	pointer  box[*item]
	external box[mail.Address]
	two      pair[string, int]
}

type invalidGeneric struct {
	argCount  box[string, int]
	notStruct list[string]
	missing   unknown[string]
	nested    box[pair[string, int]]
	noParams  plain[string]
	badPkg    box[nope.Thing]
}

type mappedGeneric struct {
	full     box[string]
	external box[mail.Address]
	fallback pair[string, int]
	nested   box[pair[string, int]]
}
