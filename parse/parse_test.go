package parse

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/teamwork/test"
	"github.com/teamwork/test/diff"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in, wantErr string
		want        *Endpoint
	}{
		{"", "", nil},
		{"Hello, world!", "", nil},
		{"Post some data", "", nil},
		{"POST /path", "", &Endpoint{Method: "POST", Path: "/path"}},
		{"POST /path tag1", "", &Endpoint{Method: "POST", Path: "/path", Tags: []string{"tag1"}}},
		{"POST /path tag1 tag2", "", &Endpoint{Method: "POST", Path: "/path", Tags: []string{"tag1", "tag2"}}},

		{"POST /path\n", "", &Endpoint{Method: "POST", Path: "/path"}},
		{"POST /path\n\n", "", &Endpoint{Method: "POST", Path: "/path"}},
		{"POST /path\n \n", "", &Endpoint{Method: "POST", Path: "/path"}},
		{"POST /path\nTagline!", "", &Endpoint{Method: "POST", Path: "/path", Tagline: "Tagline!"}},

		{"POST /path\n\nRequest body (foo): w00t", "", &Endpoint{Method: "POST", Path: "/path"}},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			out, err := Parse(tc.in)
			if !test.ErrorContains(err, tc.wantErr) {
				t.Fatalf("wrong err\nout:  %#v\nwant: %#v\n", err, tc.wantErr)
			}
			if !reflect.DeepEqual(tc.want, out) {
				t.Errorf("\n%v", diff.Diff(tc.want, out))
			}
		})
	}
}

func TestGetBlocks(t *testing.T) {
	cases := []struct {
		in      string
		wantErr string
		want    map[string]string
	}{
		{"", "", map[string]string{}},
		{"Request body:\n", `no content for header "Request body:"`, nil},

		{"Request body:\n hello\n world", "", map[string]string{
			"Request body:": " hello\n world",
		}},
		{"Well, this is\na description\n", "", map[string]string{
			"desc": "Well, this is\na description",
		}},

		{`
Well, this is
a description

Request body (text/plain):
 hello
 world

Query:
 foo

`, "", map[string]string{
			"desc": "Well, this is\na description",
			"Request body (text/plain):": " hello\n world",
			"Query:":                     " foo",
		}},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out, err := getBlocks(tc.in)
			if !test.ErrorContains(err, tc.wantErr) {
				t.Fatalf("wrong err\nout:  %#v\nwant: %#v\n", err, tc.wantErr)
			}
			if !reflect.DeepEqual(tc.want, out) {
				t.Errorf("\n%v", diff.Diff(tc.want, out))
			}
		})
	}
}

func TestParseParams(t *testing.T) {
	cases := []struct {
		in      string
		want    Param
		wantErr string
	}{
		{"hello", Param{name: "hello"}, ""},
		{"hello (string)", Param{name: "hello", kind: "string"}, ""},
		{"hello (string, required)", Param{name: "hello", kind: "string", required: true}, ""},
		{"hello: a desc", Param{name: "hello", info: "a desc"}, ""},
		{"hello: a desc (string, required)",
			Param{name: "hello", kind: "string", required: true, info: "a desc"},
			""},
		{"hello  :     a desc    (string, required)",
			Param{name: "hello", kind: "string", required: true, info: "a desc"},
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
			want := []Param{}
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
