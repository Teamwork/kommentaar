package docparse

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/teamwork/test"
	"github.com/teamwork/test/diff"
)

func TestParseComments(t *testing.T) {
	stdResp := map[int]Response{200: Response{
		ContentType: "application/json",
		Body:        &Params{Description: "OK", Params: []Param{{Name: "$empty"}}},
	}}

	cases := []struct {
		in, wantErr string
		want        *Endpoint
	}{
		// Query
		{`
			POST /path

			Query:
			  foo: hello

			Response 200: $empty
		`, "", &Endpoint{Method: "POST", Path: "/path", Request: Request{
			Query: &Params{Params: []Param{{
				Name: "foo",
				Info: "hello",
			}}},
		}}},

		// Path, Query and Form
		{`
			POST /path

			Response 200: $empty
			Query:
				foo: hello
			Form:
				Hello: WORLD {required}
		`, "", &Endpoint{
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

		{`
			POST /path

			Request body: $ref: net/mail.Address
			Response 200: $empty
		`, "", &Endpoint{Method: "POST", Path: "/path", Request: Request{
			ContentType: "application/json",
			Body:        &Params{Reference: "mail.Address"},
		}}},

		{
			`
				POST /path

				Request body (foo): $ref: net/mail.Address
				Response 200: $empty
			`, "", &Endpoint{Method: "POST", Path: "/path", Request: Request{
				ContentType: "foo",
				Body:        &Params{Reference: "mail.Address"},
			}}},

		// Two responses
		{
			`
				POST /path

				Response: $empty
				Response 400 (w00t): $empty
			`, "", &Endpoint{
				Method: "POST", Path: "/path",
				Responses: map[int]Response{
					200: {
						ContentType: "application/json",
						Body:        &Params{Description: "OK", Params: []Param{{Name: "$empty"}}},
					},
					400: {
						ContentType: "w00t",
						Body:        &Params{Description: "Bad Request", Params: []Param{{Name: "$empty"}}},
					},
				}}},

		// Duplicate response codes
		{
			`
				POST /path

				Response 200: $empty
				Response 200: $empty
			`, "response", nil},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			prog := NewProgram(false)

			if tc.want != nil && tc.want.Responses == nil {
				tc.want.Responses = stdResp
			}
			tc.in = test.NormalizeIndent(tc.in)

			out, err := ParseComment(prog, tc.in, ".", "docparse.go")
			if !test.ErrorContains(err, tc.wantErr) {
				t.Fatalf("wrong err\nout:  %#v\nwant: %#v\n", err, tc.wantErr)
			}
			if !reflect.DeepEqual(tc.want, out) {
				t.Errorf("\n%v", diff.Diff(tc.want, out))
			}
		})
	}
}

func TestGetStartLine(t *testing.T) {
	cases := []struct {
		in, wantMethod, wantPath string
		wantTags                 []string
	}{
		// Valid
		{"POST /path", "POST", "/path", nil},
		{"GET /path tag1", "GET", "/path", []string{"tag1"}},
		{"DELETE /path/str tag1 tag2", "DELETE", "/path/str", []string{"tag1", "tag2"}},
		{"PATCH /path/{id}/{var}/x tag1 tag2", "PATCH", "/path/{id}/{var}/x", []string{"tag1", "tag2"}},

		// Invalid start lines.
		{"", "", "", nil},
		{"Hello, world!", "", "", nil},
		{"Post some data", "", "", nil},
		{"POST path", "", "", nil},
		{"POST p/ath", "", "", nil},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			method, path, tags := getStartLine(tc.in)

			if method != tc.wantMethod {
				t.Errorf("method wrong\nout:  %#v\nwant: %#v\n",
					method, tc.wantMethod)
			}

			if path != tc.wantPath {
				t.Errorf("path wrong\nout:  %#v\nwant: %#v\n",
					path, tc.wantPath)
			}

			if !reflect.DeepEqual(tc.wantTags, tags) {
				t.Errorf("tags wrong\nout:  %#v\nwant: %#v\n", tags, tc.wantTags)
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
		{"Request body:\n hello\n world\nRequest body:\n hello\n world\n", "duplicate header", nil},

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

		// Single-line blocks
		{`
ANOTHER FILE!

Request body: $ref: net/mail.Address
Response 200: $ref: AnObject
Response 400: $ref: ErrorObject
Response 401: $ref: exampleimport.Foo`, "", map[string]string{
			"desc":          "ANOTHER FILE!",
			"Request body:": "$ref: net/mail.Address",
			"Response 200:": "$ref: AnObject",
			"Response 400:": "$ref: ErrorObject",
			"Response 401:": "$ref: exampleimport.Foo",
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
		{"hello: {int} {required}", Param{Name: "hello", Kind: "int", Required: true}, ""},
		{"Hello {enum: one two three}", Param{Name: "Hello", Kind: "enum", KindEnum: []string{"one", "two", "three"}}, ""},

		{"subject: The subject {required, pattern: [a-z]}", Param{}, "unknown parameter tag"},
		{"subject: foo\n$ref: testObject", Param{}, "both a reference and parameters are given"},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			prog := NewProgram(false)
			out, err := parseParams(prog, tc.in, ".")
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

			prog := NewProgram(false)
			out, err := parseParams(prog, in, ".")
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

func TestParseParamsTags(t *testing.T) {
	cases := []struct {
		in, wantLine string
		wantTags     []string
	}{
		{"", "", nil},
		{"hello", "hello", nil},
		{"hello {}", "hello", nil},
		{"hello {  }", "hello", nil},
		{"hello {int}", "hello", []string{"int"}},
		{"hello {int, required}", "hello", []string{"int", "required"}},
		{"hello {int, required,}", "hello", []string{"int", "required"}},
		{"Hello {int}{required} world", "Hello world", []string{"int", "required"}},
		{"Hello {int} world {required}", "Hello world", []string{"int", "required"}},
		{"Hello {int} {required} world", "Hello world", []string{"int", "required"}},
		{"hello {  } { } world", "hello world", nil},
		{"Hello there {int}.", "Hello there.", []string{"int"}},
		{"Hello {enum: one two three}", "Hello", []string{"enum: one two three"}},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			outLine, outTags := ParseParamsTags(tc.in)
			if outLine != tc.wantLine {
				t.Errorf("\nout:  %#v\nwant: %#v\n", outLine, tc.wantLine)
			}

			if !reflect.DeepEqual(tc.wantTags, outTags) {
				t.Errorf("\nout:  %#v\nwant: %#v\n", outTags, tc.wantTags)
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
			Package: "github.com/teamwork/kommentaar/docparse",
			File:    "", // TODO
			Lookup:  "docparse.testObject",
			Info:    "testObject general documentation.",
			Params: []Param{
				{Name: "ID", Info: "ID documentation.", Required: true},
				{Name: "Foo", Info: "Foo is a really cool foo-thing!\nSuch foo!"},
				{Name: "Bar"},
			},
		}},
		{"net/mail.Address", "", &Reference{
			Name:    "Address",
			Package: "net/mail",
			File:    "", // TODO
			Lookup:  "mail.Address",
			Info: "Address represents a single mail address.\n" +
				"An address such as \"Barry Gibbs <bg@example.com>\" is represented\n" +
				`as Address{Name: "Barry Gibbs", Address: "bg@example.com"}.`,
			Params: []Param{
				{Name: "Name", Info: "Proper name; may be empty."},
				{Name: "Address", Info: "user@domain"},
			},
		}},

		{"UnknownObject", "could not find", nil},
		{"net/http.Header", "not a struct", nil},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.in), func(t *testing.T) {
			prog := NewProgram(false)
			out, err := GetReference(prog, tc.in, ".")
			if !test.ErrorContains(err, tc.wantErr) {
				t.Fatalf("wrong err\nout:  %#v\nwant: %#v\n", err, tc.wantErr)
			}

			if out != nil {
				out.File = "" // TODO: test this as well.
			}

			if out != nil && out.Params != nil {
				for i := range out.Params {
					out.Params[i].KindField = nil
				}
			}

			if !reflect.DeepEqual(tc.want, out) {
				t.Errorf("\n%v", diff.Diff(tc.want, out))
			}
		})
	}
}
