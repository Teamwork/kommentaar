package docparse

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"testing"

	"github.com/teamwork/test/diff"
)

func TestFieldToProperty(t *testing.T) {
	want := map[string]*Schema{
		"str":       {Type: "string", Description: "Documented str field.\nNewline."},
		"byt":       {Type: "string"},
		"r":         {Type: "string"},
		"b":         {Type: "boolean", Description: "Inline docs."},
		"fl":        {Type: "number"},
		"err":       {Type: "string"},
		"strP":      {Type: "string"},
		"slice":     {Type: "array", Items: &Schema{Type: "string"}},
		"sliceP":    {Type: "array", Items: &Schema{Type: "string"}},
		"cstr":      {Type: "string"},
		"cstrP":     {Type: "string"},
		"enumStr":   {Type: "string", Enum: []string{"a", "b", "c"}},
		"enumsStr":  {Type: "array", Items: &Schema{Type: "string", Enum: []string{"a", "b", "c"}}},
		"bar":       {Reference: "a.bar"},
		"barP":      {Reference: "a.bar"},
		"pkg":       {Reference: "mail.Address"},
		"pkgSlice":  {Type: "array", Items: &Schema{Reference: "mail.Address"}},
		"pkgSliceP": {Type: "array", Items: &Schema{Reference: "mail.Address"}},
		"cSlice":    {Type: "array", Items: &Schema{Type: "string"}},
		"deeper":    {Reference: "a.refAnother"},
		"docs": {Type: "string", Description: "This has some documentation!",
			Required: []string{"docs"},
			Enum:     []string{"one", "two", "three", "four", "five", "six", "seven"},
		},
	}

	build.Default.GOPATH = "./testdata"
	ts, _, _, err := findType("./testdata/src/a/a.go", "a", "foo")
	if err != nil {
		t.Fatalf("could not parse file: %v", err)
	}

	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		t.Fatal("not a struct?!")
	}

	for i, f := range st.Fields.List {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			prog := NewProgram(false)
			out, err := fieldToSchema(prog, f.Names[0].Name, "json", Reference{
				Package: "a",
				File:    "./testdata/src/a/a.go",
				Context: "req",
			}, f, nil)
			if err != nil {
				t.Fatal(err)
			}

			for _, name := range f.Names {
				t.Run(name.Name, func(t *testing.T) {
					w, ok := want[name.Name]
					if !ok {
						t.Fatalf("no test case for %v", name)
					}

					if d := diff.Diff(w, out); d != "" {
						t.Errorf("%v", d)
					}
				})
			}
		})
	}

	t.Run("mapped types", func(t *testing.T) {
		cases := []struct {
			name     string
			mapTypes map[string]string
			want     map[string]*Schema
		}{
			{
				name: "short selector key",
				mapTypes: map[string]string{
					"mail.Address": "string",
					"a.bar":        "string",
				},
				want: map[string]*Schema{
					"b":        {Type: "string"},
					"bSlice":   {Type: "array", Items: &Schema{Type: "string"}},
					"pkg":      {Type: "string"},
					"pkgSlice": {Type: "array", Items: &Schema{Type: "string"}},
				},
			},
			{
				name: "fully-qualified key",
				mapTypes: map[string]string{
					"net/mail.Address": "string",
					"a.bar":            "string",
				},
				want: map[string]*Schema{
					"b":        {Type: "string"},
					"bSlice":   {Type: "array", Items: &Schema{Type: "string"}},
					"pkg":      {Type: "string"},
					"pkgSlice": {Type: "array", Items: &Schema{Type: "string"}},
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				ts, _, _, err := findType("./testdata/src/a/a.go", "a", "mapped")
				if err != nil {
					t.Fatalf("could not parse file: %v", err)
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					t.Fatal("not a struct?!")
				}

				prog := NewProgram(false)
				prog.Config.MapTypes = tc.mapTypes

				for _, f := range st.Fields.List {
					name := f.Names[0].Name
					out, err := fieldToSchema(prog, name, "json", Reference{
						Package: "a",
						File:    "./testdata/src/a/a.go",
						Context: "req",
					}, f, nil)
					if err != nil {
						t.Fatalf("%s: %v", name, err)
					}
					w, ok := tc.want[name]
					if !ok {
						t.Fatalf("no expected schema for %s", name)
					}
					if d := diff.Diff(w, out); d != "" {
						t.Errorf("%s: %v", name, d)
					}
				}
			})
		}
	})

	t.Run("external_enum", func(t *testing.T) {
		wantExternal := map[string]*Schema{
			"status":   {Type: "string", Enum: []string{"active", "inactive", "pending"}},
			"statuses": {Type: "array", Items: &Schema{Type: "string", Enum: []string{"active", "inactive", "pending"}}},
		}

		prog := NewProgram(false)
		ts, _, _, err := findType("./testdata/src/a/a.go", "a", "withExternalEnum")
		if err != nil {
			t.Fatalf("could not parse file: %v", err)
		}

		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			t.Fatal("not a struct?!")
		}

		for _, f := range st.Fields.List {
			out, err := fieldToSchema(prog, f.Names[0].Name, "json", Reference{
				Package: "a",
				File:    "./testdata/src/a/a.go",
				Context: "req",
			}, f, nil)
			if err != nil {
				t.Fatal(err)
			}

			for _, n := range f.Names {
				w, ok := wantExternal[n.Name]
				if !ok {
					t.Fatalf("no test case for %v", n.Name)
				}
				if d := diff.Diff(w, out); d != "" {
					t.Errorf("%v: %v", n.Name, d)
				}
			}
		}
	})

	t.Run("nested", func(t *testing.T) {
		prog := NewProgram(false)
		ts, _, _, err := findType("./testdata/src/a/a.go", "a", "nested")
		if err != nil {
			t.Fatalf("could not parse file: %v", err)
		}

		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			t.Fatal("not a struct?!")
		}

		for _, f := range st.Fields.List {
			out, err := fieldToSchema(prog, f.Names[0].Name, "json", Reference{
				Package: "a",
				File:    "./testdata/src/a/a.go",
			}, f, nil)
			if err != nil {
				t.Fatal(err)
			}

			// TODO: test.
			_ = out
		}
	})
}

func TestIsInferredRequired(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want bool
	}{
		{"non-pointer no tag", `struct{ F string }`, true},
		{"non-pointer json tag", "struct{ F string `json:\"f\"` }", true},
		{"pointer", `struct{ F *string }`, false},
		{"omitempty", "struct{ F string `json:\"f,omitempty\"` }", false},
		{"omitempty with whitespace", "struct{ F string `json:\"f, omitempty\"` }", false},
		{"other tag option not omitempty", "struct{ F string `json:\"f,string\"` }", true},
		{"explicit optional doc", "struct{\n// {optional}\nF string\n}", false},
		{"explicit required on pointer doc", "struct{\n// {required}\nF *string\n}", false},
		{"embedded", `struct{ string }`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			src := "package p\ntype T " + tc.src
			file, err := parser.ParseFile(fset, "in.go", src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			st := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.StructType)
			got := isInferredRequired(st.Fields.List[0], "json")
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
