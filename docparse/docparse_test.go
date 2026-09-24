package docparse

import (
	"fmt"
	"go/build"
	"reflect"
	"testing"

	"github.com/teamwork/test"
	"github.com/teamwork/test/diff"
)

func TestParseComments(t *testing.T) {
	stdResp := map[int]Response{200: {
		ContentType: "application/json",
		Body:        &Ref{Description: "200 OK (no data)"},
	}}

	tests := []struct {
		name        string
		in, wantErr string
		want        []*Endpoint
	}{
		{"tagline", `
POST /path
The tagline!

Request body (foo): net/mail.Address
Response 200: {empty}
			`,
			"",
			[]*Endpoint{{
				Method:  "POST",
				Path:    "/path",
				Tagline: "The tagline!",
				Request: Request{
					ContentType: "foo",
					Body:        &Ref{Reference: "mail.Address"},
				},
			}},
		},

		{"two-routes", `
POST /path
GET /foo

Request body (foo): net/mail.Address
Response 200: {empty}
			`,
			"",
			[]*Endpoint{{
				Method: "POST",
				Path:   "/path",
				Request: Request{
					ContentType: "foo",
					Body:        &Ref{Reference: "mail.Address"},
				},
			}, {
				Method: "GET",
				Path:   "/foo",
			}},
		},

		{"two-routes-tagline", `
POST /path
GET /foo
The tagline!

Request body (foo): net/mail.Address
Response 200: {empty}
			`,
			"",
			[]*Endpoint{{
				Method:  "POST",
				Path:    "/path",
				Tagline: "The tagline!",
				Request: Request{
					ContentType: "foo",
					Body:        &Ref{Reference: "mail.Address"},
				},
			}, {
				Method: "GET",
				Path:   "/foo",
			}},
		},

		{"two-routes-tagline-desc", `
POST /path
GET /foo
The tagline!

Some desc!

Request body (foo): net/mail.Address
Response 200: {empty}
			`,
			"",
			[]*Endpoint{{
				Method:  "POST",
				Path:    "/path",
				Tagline: "The tagline!",
				Info:    "Some desc!",
				Request: Request{
					ContentType: "foo",
					Body:        &Ref{Reference: "mail.Address"},
				},
			}, {
				Method: "GET",
				Path:   "/foo",
			}},
		},

		{"single-desc", `
POST /path

A description.

Request body (foo): net/mail.Address
Response 200: {empty}
			`,
			"",
			[]*Endpoint{{
				Method: "POST",
				Path:   "/path",
				Info:   "A description.",
				Request: Request{
					ContentType: "foo",
					Body:        &Ref{Reference: "mail.Address"},
				},
			}},
		},

		{"multi-desc", `
POST /path

A description.
Of multiple lines.

Request body (foo): net/mail.Address
Response 200: {empty}
			`,
			"",
			[]*Endpoint{{
				Method: "POST",
				Path:   "/path",
				Info:   "A description.\nOf multiple lines.",
				Request: Request{
					ContentType: "foo",
					Body:        &Ref{Reference: "mail.Address"},
				},
			}},
		},

		{"multi-desc-with-empty-lines", `
POST /path

A description.
Of multiple lines.

With some more.

And some more.

Request body (foo): net/mail.Address
Response 200: {empty}
			`,
			"",
			[]*Endpoint{{
				Method: "POST",
				Path:   "/path",
				Info:   "A description.\nOf multiple lines.\n\nWith some more.\n\nAnd some more.",
				Request: Request{
					ContentType: "foo",
					Body:        &Ref{Reference: "mail.Address"},
				},
			}},
		},

		{"single-desc-and-tagline", `
POST /path
The tagline!

A description.

Request body (foo): net/mail.Address
Response 200: {empty}
			`,
			"",
			[]*Endpoint{{
				Method:  "POST",
				Path:    "/path",
				Tagline: "The tagline!",
				Info:    "A description.",
				Request: Request{
					ContentType: "foo",
					Body:        &Ref{Reference: "mail.Address"},
				},
			}},
		},

		{"multi-desc-and-tagline", `
POST /path
The tagline!

A description.
Of multiple lines.

Request body (foo): net/mail.Address
Response 200: {empty}
			`,
			"",
			[]*Endpoint{{
				Method:  "POST",
				Path:    "/path",
				Tagline: "The tagline!",
				Info:    "A description.\nOf multiple lines.",
				Request: Request{
					ContentType: "foo",
					Body:        &Ref{Reference: "mail.Address"},
				},
			}},
		},

		{"req-ref", `
POST /path

Request body: net/mail.Address
Response 200: {empty}
		`,
			"",
			[]*Endpoint{{
				Method: "POST",
				Path:   "/path",
				Request: Request{
					ContentType: "application/json",
					Body:        &Ref{Reference: "mail.Address"},
				}},
			}},

		{"path-ref", `
POST /path/{Name}/{Address}

Path: net/mail.Address
Response 200: {empty}
		`,
			"",
			[]*Endpoint{{
				Method: "POST",
				Path:   "/path/{Name}/{Address}",
				Request: Request{
					Path: &Ref{Reference: "mail.Address"},
				}},
			}},

		{"req-content-type", `
POST /path

Request body (foo): net/mail.Address
Response 200: {empty}
			`,
			"",
			[]*Endpoint{{
				Method: "POST",
				Path:   "/path",
				Request: Request{
					ContentType: "foo",
					Body:        &Ref{Reference: "mail.Address"},
				},
			}},
		},

		{"response-ref", `
POST /path

Response: {empty}
Response 400 (w00t): {empty}
			`,
			"",
			[]*Endpoint{{
				Method: "POST",
				Path:   "/path",
				Responses: map[int]Response{
					200: {
						ContentType: "application/json",
						Body:        &Ref{Description: "200 OK (no data)"},
					},
					400: {
						ContentType: "w00t",
						Body:        &Ref{Description: "400 Bad Request (no data)"},
					},
				},
			}},
		},

		//	{"err-double-code", `
		//		POST /path

		//		Response 200: {empty}
		//		Response 200: {empty}
		//	`, "response", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := NewProgram(false)

			if tt.want != nil && tt.want[0].Responses == nil {
				tt.want[0].Responses = stdResp
			}
			tt.in = test.NormalizeIndent(tt.in)

			out, _, err := parseComment(prog, tt.in, ".", "docparse.go")
			if !test.ErrorContains(err, tt.wantErr) {
				t.Fatalf("wrong err\nout:  %#v\nwant: %#v\n", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.want, out) {
				t.Errorf("\n%v", diff.Diff(tt.want, out))
			}
		})
	}
}

func TestGetStartLine(t *testing.T) {
	tests := []struct {
		in, wantMethod, wantPath string
		wantTags                 []string
	}{
		// Valid
		{"POST /path",
			"POST", "/path", nil},
		{"GET /path tag1",
			"GET", "/path", []string{"tag1"}},
		{"DELETE /path/str tag1 tag2",
			"DELETE", "/path/str", []string{"tag1", "tag2"}},
		{"PATCH /path/{id}/{var}/x tag1 tag2",
			"PATCH", "/path/{id}/{var}/x", []string{"tag1", "tag2"}},

		// Invalid start lines.
		{"", "", "", nil},
		{"Hello, world!", "", "", nil},
		{"Post some data", "", "", nil},
		{"POST path", "", "", nil},
		{"POST p/ath", "", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			method, path, tags := parseStartLine(tt.in)

			if method != tt.wantMethod {
				t.Errorf("method wrong\nout:  %#v\nwant: %#v\n",
					method, tt.wantMethod)
			}

			if path != tt.wantPath {
				t.Errorf("path wrong\nout:  %#v\nwant: %#v\n",
					path, tt.wantPath)
			}

			if !reflect.DeepEqual(tt.wantTags, tags) {
				t.Errorf("tags wrong\nout:  %#v\nwant: %#v\n", tags, tt.wantTags)
			}
		})
	}
}

/*
func TestParseParams(t *testing.T) {
	tests := []struct {
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

		{"subject: The subject {required, pattern: [a-z]}", Param{}, "unknown parameter property"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			prog := NewProgram(false)
			out, err := parseParams(prog, tt.in, ".")
			if !test.ErrorContains(err, tt.wantErr) {
				t.Fatalf("wrong err\nout:  %#v\nwant: %#v\n", err, tt.wantErr)
			}
			if tt.wantErr == "" && !reflect.DeepEqual(tt.want, out.Params[0]) {
				t.Errorf("\nout:  %#v\nwant: %#v\n", out.Params[0], tt.want)
			}
		})
	}

	if !t.Failed() {
		t.Run("combined", func(t *testing.T) {
			in := ""
			want := []Param{}
			for _, tt := range tests {
				if tt.wantErr == "" {
					in += tt.in + "\n"
					want = append(want, tt.want)
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
*/

func TestParseParamsTags(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			outLine, outTags := parseTags(tt.in)
			if outLine != tt.wantLine {
				t.Errorf("\nout:  %#v\nwant: %#v\n", outLine, tt.wantLine)
			}

			if !reflect.DeepEqual(tt.wantTags, outTags) {
				t.Errorf("\nout:  %#v\nwant: %#v\n", outTags, tt.wantTags)
			}
		})
	}
}

func TestGetReference(t *testing.T) {
	tests := []struct {
		in      string
		wantErr string
		want    *Reference
	}{
		{"testObject", "", &Reference{
			Name:    "testObject",
			Package: "github.com/teamwork/kommentaar/docparse",
			File:    "", // TODO
			Lookup:  "docparse.testObject",
			Context: "req",
			Info:    "testObject general documentation.",
			Fields: []Param{
				{Name: "ID"},
				{Name: "Foo"},
				{Name: "Bar"},
			},
			Schema: &Schema{
				Title:       "testObject",
				Description: "testObject general documentation.",
				Type:        "object",
				Required:    []string{"ID"},
				Properties: map[string]*Schema{
					"ID":  {Type: "integer", Description: "ID documentation."},
					"Foo": {Type: "string", Description: "Foo is a really cool foo-thing!\nSuch foo!"},
					"Bar": {Type: "array", Items: &Schema{Type: "string"}},
				},
			},
		}},
		{"net/mail.Address", "", &Reference{
			Name:    "Address",
			Package: "net/mail",
			File:    "", // TODO
			Lookup:  "mail.Address",
			Context: "req",
			Info: "Address represents a single mail address.\n" +
				"An address such as \"Barry Gibbs <bg@example.com>\" is represented\n" +
				`as Address{Name: "Barry Gibbs", Address: "bg@example.com"}.`,
			Fields: []Param{
				{Name: "Name"},
				{Name: "Address"},
			},
			Schema: &Schema{
				Title: "Address",
				Description: "Address represents a single mail address.\n" +
					"An address such as \"Barry Gibbs <bg@example.com>\" is represented\n" +
					"as Address{Name: \"Barry Gibbs\", Address: \"bg@example.com\"}.",
				Type: "object",
				Properties: map[string]*Schema{
					"Address": {Type: "string", Description: "user@domain"},
					"Name":    {Type: "string", Description: "Proper name; may be empty."},
				},
			},
		}},

		{"UnknownObject", "could not find", nil},
		{"net/http.Header", "", &Reference{
			Name:    "Header",
			Package: "net/http",
			Lookup:  "http.Header",
			Info: "A Header represents the key-value pairs in an HTTP header.\n\n" +
				"The keys should be in canonical form, as returned by\n" +
				"[CanonicalHeaderKey].",
			Context: "req",
			Schema: &Schema{
				Title: "Header",
				Description: "A Header represents the key-value pairs in an HTTP header.\n\n" +
					"The keys should be in canonical form, as returned by\n" +
					"[CanonicalHeaderKey].",
				Type:       "object",
				Properties: map[string]*Schema{},
			},
		}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.in), func(t *testing.T) {
			prog := NewProgram(false)
			out, err := GetReference(prog, "req", false, tt.in, ".")
			if !test.ErrorContains(err, tt.wantErr) {
				t.Fatalf("wrong err\nout:  %v\nwant: %v\n", err, tt.wantErr)
			}

			if out != nil {
				out.File = "" // TODO: test this as well.
			}

			if out != nil && out.Fields != nil {
				for i := range out.Fields {
					out.Fields[i].KindField = nil
				}
			}

			if !reflect.DeepEqual(tt.want, out) {
				t.Errorf("\n%v", diff.Diff(tt.want, out))
			}
		})
	}
}

// Regression: an embedded qualified pointer type tagged json:"-" (for
// example *mail.Address with a json:"-" tag) previously caused a
// nil-pointer panic because the first field-walk pass tried to resolve
// the embed before checking the tag. The fix is to honor json:"-" in
// that pass too. We also verify the embed is absent from the resulting
// Reference.Fields and the referenced external type is not pulled into
// prog.References.
func TestGetReference_EmbedJSONDash(t *testing.T) {
	prog := NewProgram(false)
	prog.Config.StructTag = "json"
	out, err := GetReference(prog, "req", false, "testEmbedJSONDash", ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out == nil {
		t.Fatal("nil reference")
	}

	var names []string
	for _, f := range out.Fields {
		names = append(names, f.Name)
	}
	wantFields := []string{"ID"}
	if !reflect.DeepEqual(names, wantFields) {
		t.Errorf("fields: got %v, want %v", names, wantFields)
	}

	if _, ok := prog.References["mail.Address"]; ok {
		t.Errorf("mail.Address was resolved despite json:\"-\" on embed")
	}
}

func TestParseResponse(t *testing.T) {
	tests := []struct {
		in       string
		wantCode int
		wantResp *Response
		wantErr  string
	}{
		{
			"Response 400: net/mail.Address",
			400,
			&Response{
				ContentType: "application/json",
				Body:        &Ref{Reference: "mail.Address", Description: "400 Bad Request"},
			},
			"",
		},
		{
			"Response 400 net/mail.Address",
			0,
			nil,
			"",
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			prog := NewProgram(false)
			code, resp, err := ParseResponse(prog, "", tt.in)

			if !test.ErrorContains(err, tt.wantErr) {
				t.Fatalf("wrong error\nwant: %v\ngot:  %v", tt.wantErr, err)
			}
			if code != tt.wantCode {
				t.Errorf("wrong code\nwant: %v\ngot:  %v", tt.wantCode, code)
			}
			if d := diff.Diff(tt.wantResp, resp); d != "" {
				t.Error(d)
			}
		})
	}
}

// TestGetReferencePackageCollision makes sure GetReference does not return
// another package's definition when two files each name the same base
// package. Package repa/report and package repb/report both declare a type
// Nested and both have the package name "report", and testdata/src/g and
// testdata/src/h each import one of them unaliased, so both files ask
// GetReference for the same string: "report.Nested".
func TestGetReferencePackageCollision(t *testing.T) {
	orig := build.Default.GOPATH
	build.Default.GOPATH = "./testdata"
	defer func() { build.Default.GOPATH = orig }()
	prog := NewProgram(false)

	first, err := GetReference(prog, "req", false, "report.Nested", "./testdata/src/g/g.go")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := GetReference(prog, "resp", false, "report.Nested", "./testdata/src/h/h.go")
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if first.Package != "repa/report" {
		t.Errorf("first.Package = %q, want %q", first.Package, "repa/report")
	}
	if second.Package != "repb/report" {
		t.Errorf("second.Package = %q, want %q", second.Package, "repb/report")
	}
	if first.Lookup == second.Lookup {
		t.Errorf("first and second both resolved to %q, want different keys", first.Lookup)
	}

	wantFirst := []string{"Str"}
	wantSecond := []string{"Num"}
	if got := fieldNames(first.Fields); !reflect.DeepEqual(got, wantFirst) {
		t.Errorf("first fields = %v, want %v", got, wantFirst)
	}
	if got := fieldNames(second.Fields); !reflect.DeepEqual(got, wantSecond) {
		t.Errorf("second fields = %v, want %v", got, wantSecond)
	}

	if stored, ok := prog.References[first.Lookup]; !ok || stored.Package != "repa/report" {
		t.Errorf("prog.References[%q] = %+v, want Package %q", first.Lookup, stored, "repa/report")
	}
	if stored, ok := prog.References[second.Lookup]; !ok || stored.Package != "repb/report" {
		t.Errorf("prog.References[%q] = %+v, want Package %q", second.Lookup, stored, "repb/report")
	}

	// A second call with the same raw lookup as the first must still hit the
	// first package, not fall through to the second.
	again, err := GetReference(prog, "req", false, "report.Nested", "./testdata/src/g/g.go")
	if err != nil {
		t.Fatalf("again: %v", err)
	}
	if again.Lookup != first.Lookup || again.Package != "repa/report" {
		t.Errorf("again = %+v, want the same as first", again)
	}
}

// TestGetReferenceDottedSlice makes sure GetReference resolves a named slice
// type in a package whose import path has a dot.
func TestGetReferenceDottedSlice(t *testing.T) {
	orig := build.Default.GOPATH
	build.Default.GOPATH = "./testdata"
	defer func() { build.Default.GOPATH = orig }()
	prog := NewProgram(false)

	out, err := GetReference(prog, "req", false, "example.com/m.Items", "./testdata/src/a/a.go")
	if err != nil {
		t.Fatal(err)
	}
	if out.Lookup != "m.Item" || out.Package != "example.com/m" || !out.IsSlice {
		t.Errorf("out = %+v, want the slice of m.Item in example.com/m", out)
	}
}
