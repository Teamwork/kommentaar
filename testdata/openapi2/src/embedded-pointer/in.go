package req

// resp docs.
type resp struct {
	*other `json:"o"`
}

// other docs.
type other struct {
	Other string `json:"other"` // Other.
}

// POST /path
//
// Response 200: resp
