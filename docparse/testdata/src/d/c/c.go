package c

type Nested struct {
	num int
}

// Tree holds a map of itself.
type Tree struct {
	Children map[string]Tree `json:"children"`
}
