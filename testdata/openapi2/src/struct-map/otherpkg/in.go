package otherpkg

// No docs from here, it doesn't have its own definition.
type MyMap map[int64]map[string]interface{}

// OtherStruct is a struct in another package.
type OtherStruct struct {
	// Map contains some random data :)
	Map MyMap `json:"map"`
}
