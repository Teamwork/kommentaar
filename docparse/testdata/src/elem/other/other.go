package other

type Rows []struct {
	Item Item `json:"item"`
}

type Item struct {
	ID int `json:"id"`
}
