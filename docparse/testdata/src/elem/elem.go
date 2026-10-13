package elem

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

	ch []chan int
}
