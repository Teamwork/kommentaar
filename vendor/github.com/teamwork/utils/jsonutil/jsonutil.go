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

// MustUnmarshal behaves like json.Unmarshal but will panic on errors.
func MustUnmarshal(data []byte, v interface{}) {
	err := json.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}
}
