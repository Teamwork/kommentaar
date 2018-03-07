package docparse

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/teamwork/test"
	"github.com/teamwork/test/diff"
)

var (
	stdResp = "Response 200:\n  nodoc\n"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in, wantErr string
		want        *Endpoint
	}{
		// Not valid start lines.
		{"", "", nil},
		{"Hello, world!", "", nil},
		{"Post some data", "", nil},
		{"POST path", "", nil},
		{"POST p/ath", "", nil},

		// Valid start lines with tags
		{"POST /path", "", &Endpoint{Method: "POST", Path: "/path"}},
		{"POST /path tag1", "", &Endpoint{Method: "POST", Path: "/path", Tags: []string{"tag1"}}},
		{"POST /path tag1 tag2", "", &Endpoint{Method: "POST", Path: "/path", Tags: []string{"tag1", "tag2"}}},

		// Valid start lines with tagline/description.
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
			Query: &Params{Params: []Param{{
				Name: "foo",
				Info: "hello",
			}}},
		}}},
		{"POST /path/:foo\n\nPath:\n  foo: hello", "", &Endpoint{Method: "POST", Path: "/path/:foo", Request: Request{
			Path: &Params{Params: []Param{{
				Name: "foo",
				Info: "hello",
			}}},
		}}},

		// Path, Query and Form
		{"POST /path\n\nQuery:\n  foo: hello\nForm:\n  Hello: WORLD {required}", "", &Endpoint{
			Method: "POST", Path: "/path", Request: Request{
				Query: &Params{Params: []Param{{
					Name: "foo",
					Info: "hello",
				}}},
				Form: &Params{Params: []Param{{
					Name:     "Hello",
					Info:     "WORLD",
					Required: true,
				}}},
			}}},

		{"POST /path\n\nRequest body:\n w00t", "", &Endpoint{Method: "POST", Path: "/path", Request: Request{
			ContentType: "application/json",
			Body:        &Params{Params: []Param{{Name: "w00t"}}},
		}}},

		{"POST /path\n\nRequest body (foo):\n w00t", "", &Endpoint{Method: "POST", Path: "/path", Request: Request{
			ContentType: "foo",
			Body:        &Params{Params: []Param{{Name: "w00t"}}},
		}}},

		// Single response
		{"POST /path\n\nResponse:\n w00t", "", &Endpoint{Method: "POST", Path: "/path",
			Responses: map[int]Response{
				200: {
					ContentType: "application/json",
					Body:        &Params{Params: []Param{{Name: "w00t"}}},
				}}}},

		// Two responses
		{"POST /path\n\nResponse:\n w00t\n\nResponse 400 (w00t):\n asd", "", &Endpoint{
			Method: "POST", Path: "/path",
			Responses: map[int]Response{
				200: {
					ContentType: "application/json",
					Body:        &Params{Params: []Param{{Name: "w00t"}}},
				},
				400: {
					ContentType: "w00t",
					Body:        &Params{Params: []Param{{Name: "asd"}}},
				},
			}}},

		// Duplicate response codes
		{"POST /path\n\nResponse 200:\n w00t\n\nResponse 200:\n w00t\n", "duplicate", nil},
	}

	InitProgram(true)
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out, err := ParseComment(tc.in, ".", ".")
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
		{"Request body:\nwoot:\n", `no content for header "Request body:"`, nil},

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
		{"hello {string}", Param{Name: "hello", Kind: "string"}, ""},
		{"hello {optional}", Param{Name: "hello"}, ""},
		{"hello {string, required}", Param{Name: "hello", Kind: "string", Required: true}, ""},
		{"hello: a desc", Param{Name: "hello", Info: "a desc"}, ""},
		{"hello: a desc {string, required}", Param{
			Name: "hello", Kind: "string", Required: true, Info: "a desc",
		}, ""},
		{"hello  :     a desc    {string, required}", Param{
			Name: "hello", Kind: "string", Required: true, Info: "a desc",
		}, ""},
		{"hello: a\n   desc\n   {string,\n   required\n   }", Param{
			Name: "hello", Info: "a desc", Kind: "string", Required: true,
		}, ""},
		{"same_format: {string, optional}", Param{Name: "same_format", Kind: "string"}, ""},
		{"subject: The subject {}", Param{Name: "subject", Info: "The subject"}, ""},

		{"subject: The subject {required, pattern: [a-z]}", Param{}, "unknown parameter tag"},
		{"subject: foo\n$object: testObject", Param{}, "both a reference and parameters are given"},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out, err := parseParams(tc.in, ".")
			if !test.ErrorContains(err, tc.wantErr) {
				t.Fatalf("wrong err\nout:  %#v\nwant: %#v\n", err, tc.wantErr)
			}
			if tc.wantErr == "" && !reflect.DeepEqual(tc.want, out.Params[0]) {
				t.Errorf("\nout:  %#v\nwant: %#v\n", out.Params[0], tc.want)
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

			out, err := parseParams(in, ".")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(want, out.Params) {
				t.Errorf("could not parse combined string:\n%v\n%v",
					in, diff.Diff(want, out.Params))
			}
		})
	}
}

func TestCollapseIndent(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"Hello", []string{"Hello"}},
		{"Hello\nWorld", []string{"Hello", "World"}},
		{"Hello\n World", []string{"Hello World"}},
		{" Hello\n\tWorld", []string{"Hello World"}},
		{"Hello\n  World", []string{"Hello World"}},
		{"Hello\n  World\n  Test", []string{"Hello World Test"}},
		{"Hello\nworld\n test", []string{"Hello", "world test"}},
		{"Hello\nworld\n test\nfoo", []string{"Hello", "world test", "foo"}},
		{"Hello\n\nworld\n test", []string{"Hello", "world test"}},
		{" Hello\n  world\n foo\n  bar\n", []string{"Hello world", "foo bar"}},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out := collapseIndents(tc.in)
			if !reflect.DeepEqual(tc.want, out) {
				t.Errorf("\nout:  %#v\nwant: %#v\n", out, tc.want)
			}
		})
	}
}

func TestGetReference(t *testing.T) {
	cases := []struct {
		in      string
		wantErr string
		want    *Reference
	}{
		{"testObject", "", &Reference{
			Name:    "testObject",
			Package: ".",
			Info:    "testObject general documentation.",
			Params: []Param{
				{Name: "ID", Kind: "int", Info: "ID documentation", Required: true},
				{Name: "Foo", Kind: "string", Info: "Foo is a really cool foo-thing! Such foo!"},
				{Name: "Bar", Kind: "[]string"},
			},
		}},
		{"net/mail.Address", "", &Reference{
			Name:    "Address",
			Package: "net/mail",
			Info: "Address represents a single mail address.\n" +
				"An address such as \"Barry Gibbs <bg@example.com>\" is represented\n" +
				`as Address{Name: "Barry Gibbs", Address: "bg@example.com"}.`,
			Params: []Param{
				{Name: "Name", Kind: "string", Info: "Proper name; may be empty."},
				{Name: "Address", Kind: "string", Info: "user@domain"},
			},
		}},

		{"UnknownObject", "could not find", nil},
		{"net/http.Header", "not a struct", nil},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.in), func(t *testing.T) {
			out, _, err := getReference(tc.in, ".")
			if !test.ErrorContains(err, tc.wantErr) {
				t.Fatalf("wrong err\nout:  %#v\nwant: %#v\n", err, tc.wantErr)
			}
			if !reflect.DeepEqual(tc.want, out) {
				t.Errorf("\n%v", diff.Diff(tc.want, out))
			}
		})
	}
}
