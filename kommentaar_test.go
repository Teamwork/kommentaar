package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/teamwork/test"
)

func TestParseParams(t *testing.T) {
	cases := []struct {
		in      string
		want    param
		wantErr string
	}{
		{"hello", param{name: "hello"}, ""},
		{"hello (string)", param{name: "hello", kind: "string"}, ""},
		{"hello (string, required)", param{name: "hello", kind: "string", required: true}, ""},
		{"hello: a desc", param{name: "hello", info: "a desc"}, ""},
		{"hello: a desc (string, required)",
			param{name: "hello", kind: "string", required: true, info: "a desc"},
			""},
		{"hello  :     a desc    (string, required)",
			param{name: "hello", kind: "string", required: true, info: "a desc"},
			""},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.in), func(t *testing.T) {
			out, err := parseParams(tc.in)
			if !test.ErrorContains(err, tc.wantErr) {
				t.Errorf("wrong err\nout:  %#v\nwant: %#v\n", err, tc.wantErr)
			}
			if !reflect.DeepEqual(tc.want, out[0]) {
				t.Errorf("\nout:  %#v\nwant: %#v\n", out[0], tc.want)
			}
		})
	}

	if !t.Failed() {
		t.Run("combined", func(t *testing.T) {
			in := ""
			want := []param{}
			for _, tc := range cases {
				if tc.wantErr == "" {
					in += tc.in + "\n"
					want = append(want, tc.want)
				}
			}

			out, err := parseParams(in)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(want, out) {
				t.Errorf("\nout:  %#v\nwant: %#v\n", out, want)
			}
		})
	}
}

func TestMakeEndpoint(t *testing.T) {
	cases := []struct {
		in, wantErr string
		want        *endpoint
	}{
		{"", "", nil},
		{},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out, err := makeEndpoint(tc.in)
			if !test.ErrorContains(err, tc.wantErr) {
				t.Errorf("wrong err\nout:  %#v\nwant: %#v\n", err, tc.wantErr)
			}
			if !reflect.DeepEqual(tc.want, out) {
				t.Errorf("\nout:  %#v\nwant: %#v\n", out, tc.want)
			}
		})
	}
}
