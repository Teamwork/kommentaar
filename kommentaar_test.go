package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseParams(t *testing.T) {
	cases := []struct {
		in   string
		want []param
	}{
		{
			"hello",
			[]param{param{name: "hello"}},
		},
		{
			"hello (string)",
			[]param{param{name: "hello", kind: "string"}},
		},
		{
			"hello (string, required)",
			[]param{param{name: "hello", kind: "string", required: true}},
		},
		{
			"hello: a desc",
			[]param{param{name: "hello", info: "a desc"}},
		},
		{
			"hello: a desc (string, required)",
			[]param{param{name: "hello", kind: "string", required: true, info: "a desc"}},
		},
		{
			"hello  :     a desc    (string, required)",
			[]param{param{name: "hello", kind: "string", required: true, info: "a desc"}},
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.in), func(t *testing.T) {
			out, err := parseParams(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.want, out) {
				t.Errorf("\nout:  %#v\nwant: %#v\n", out, tc.want)
			}
		})
	}
}
