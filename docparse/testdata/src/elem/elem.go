package elem

import "elem/other"

type item struct {
	ID int `json:"id"`
}

type box[T any] struct {
	V T `json:"v"`
}

type pair[K, V any] struct {
	K K `json:"k"`
	V V `json:"v"`
}

type grid[T any] struct {
	Rows [][]T `json:"rows"`
}

type elems struct {
	nestedRef [][]*item
	// {enum: x y}
	nestedEnum [][]string
	nestedChan [][]chan int

	mapRef  []map[string]item
	mapChan []map[string][]chan int

	iface []interface{}

	anon []struct {
		ID   int   `json:"id"`
		Item *item `json:"item"`
	}
	anonChan []struct{ C chan int }
	anonPkg  other.Rows

	box     []box[item]
	pair    []pair[string, int]
	grid    []grid[string]
	boxArgs []box[string, int]

	ch []chan int
}
