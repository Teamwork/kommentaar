package a

// Whitelist holds a Pair with only the field One.
type Whitelist struct {
	Pair Pair `json:"pair"` // {field-whitelist: one}
}

type Pair struct {
	One string `json:"one"`
	Two string `json:"two"`
}
