package path

import "struct-map/otherpkg"

type resp struct {
	Basic       map[string]interface{}            `json:"basic"`       // Basic comment.
	Basic2      map[string]any                    `json:"basic2"`      // Basic2 comment.
	Custom      myMap                             `json:"custom"`      // Custom comment.
	Struct      aStruct                           `json:"aStruct"`     // Struct comment.
	OtherStruct otherpkg.OtherStruct              `json:"otherStruct"` // OtherStruct comment.
	PrimSlices  map[string][]string               `json:"primSlices"`  // Map of primitive slices.
	StructSlice map[int64][]aStruct               `json:"structSlice"` // Map of local struct slices.
	OtherSlice  map[string][]otherpkg.OtherStruct `json:"otherSlice"`  // Map of cross-package struct slices.
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
