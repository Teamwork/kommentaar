package path

import "struct-map/otherpkg"

type resp struct {
	Basic       map[string]interface{} `json:"basic"`       // Basic comment.
	Custom      myMap                  `json:"custom"`      // Custom comment.
	Struct      aStruct                `json:"aStruct"`     // Struct comment.
	OtherStruct otherpkg.OtherStruct   `json:"otherStruct"` // OtherStruct comment.
}

// Comments are lost here as its just in the doc as an object.
type myMap map[int]string

type aStruct struct {
	Foo string         `json:"foo"`
	Bar otherpkg.MyMap `json:"bar"`
}

// POST /path
//
// Response 200: resp
