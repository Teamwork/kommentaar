package docparse

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
		{"POST /path\nTagline!\n\nDesc!\ndesc!", "", &Endpoint{
			Method: "POST", Path: "/path", Tagline: "Tagline!", Info: "Desc!\ndesc!"}},
		{"POST /path\n\nDesc!\ndesc!", "", &Endpoint{
			Method: "POST", Path: "/path", Info: "Desc!\ndesc!"}},

		// Query
		{"POST /path\n\nQuery:\n  foo: hello", "", &Endpoint{Method: "POST", Path: "/path", Request: Request{
			Query: []Param{{
				Name: "foo",
				Info: "hello",
			}},
		}}},
		{"POST /path/:foo\n\nPath:\n  foo: hello", "", &Endpoint{Method: "POST", Path: "/path/:foo", Request: Request{
			Path: []Param{{
				Name: "foo",
				Info: "hello",
			}},
		}}},
		// Path
		// Query and Form
		{"POST /path\n\nQuery:\n  foo: hello\nForm:\n  Hello: WORLD (required)", "", &Endpoint{
			Method: "POST", Path: "/path", Request: Request{
				Query: []Param{{
					Name: "foo",
					Info: "hello",
				}},
				Form: []Param{{
					Name:     "Hello",
					Info:     "WORLD",
					Required: true,
				}},
			}}},

		{"POST /path\n\nRequest body:\n w00t", "", &Endpoint{Method: "POST", Path: "/path", Request: Request{
			ContentType: defaultRequest,
		}}},

		{"POST /path\n\nRequest body (foo):\n w00t", "", &Endpoint{Method: "POST", Path: "/path", Request: Request{
			ContentType: "foo",
		}}},

		{"POST /path\n\nResponse:\n w00t", "", &Endpoint{Method: "POST", Path: "/path",
			Responses: map[int]Response{
				200: {ContentType: defaultResponse},
			}}},

		{"POST /path\n\nResponse:\n w00t\n\nResponse 400 (w00t):\n asd", "", &Endpoint{
			Method: "POST", Path: "/path",
			Responses: map[int]Response{
				200: {ContentType: defaultResponse},
				400: {ContentType: "w00t"},
			}}},
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
		{"hello", Param{Name: "hello"}, ""},
		{"hello (string)", Param{Name: "hello", Kind: "string"}, ""},
		{"hello (string, required)", Param{Name: "hello", Kind: "string", Required: true}, ""},
		{"hello: a desc", Param{Name: "hello", Info: "a desc"}, ""},
		{"hello: a desc (string, required)",
			Param{Name: "hello", Kind: "string", Required: true, Info: "a desc"},
			""},
		{"hello  :     a desc    (string, required)",
			Param{Name: "hello", Kind: "string", Required: true, Info: "a desc"},
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
