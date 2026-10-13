package elem

import "elem/other"

type item struct {
	ID int `json:"id"`
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

	ch []chan int
}
