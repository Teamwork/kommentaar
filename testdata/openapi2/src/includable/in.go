package path

// FirstStruct docs
type FirstStruct struct {
	FieldOne string `json:"field_one"`
	FieldTwo string `json:"field_two"`
}

// SecondStruct docs
type SecondStruct struct {
	FirstStruct FirstStruct `json:"firststruct" includable:"fieldtwo"`
}

// POST /path
//
// Response 200: SecondStruct
