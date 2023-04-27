// Package jsonutil provides functions for working with JSON.
package jsonutil // import "github.com/teamwork/utils/jsonutil"

import "encoding/json"

// MustMarshal behaves like json.Marshal but will panic on errors.
func MustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// MustMarshalIndent behaves like json.MarshalIndent but will panic on errors.
func MustMarshalIndent(v interface{}, prefix, indent string) []byte {
	b, err := json.MarshalIndent(v, prefix, indent)
	if err != nil {
		panic(err)
	}
	return b
}

// MustUnmarshal behaves like json.Unmarshal but will panic on errors.
func MustUnmarshal(data []byte, v interface{}) {
	err := json.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}
}

// Indent a json string by unmarshalling it and marshalling it with
// MarshalIndent.
//
// The data will be unmarshalled in to v, which must be a pointer. Example:
//
//	Indent(`{"a": "b"}`, &map[string]string{}, "", "  ")
func Indent(data []byte, v interface{}, prefix, indent string) ([]byte, error) {
	err := json.Unmarshal(data, v)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(v, prefix, indent)
}

// MustIndent behaves like Indent but will panic on errors.
func MustIndent(data []byte, v interface{}, prefix, indent string) []byte {
	b, err := Indent(data, v, prefix, indent)
	if err != nil {
		panic(err)
	}
	return b
}
