package docparse

import (
	"fmt"
	"go/ast"
	"go/build"
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
		"bar":       {Reference: "a.bar"},
		"barP":      {Reference: "a.bar"},
		"pkg":       {Reference: "mail.Address"},
		"pkgSlice":  {Type: "array", Items: &Schema{Reference: "mail.Address"}},
		"pkgSliceP": {Type: "array", Items: &Schema{Reference: "mail.Address"}},
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
			}, f)
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
			}, f)
			if err != nil {
				t.Fatal(err)
			}

			// TODO: test.
			_ = out
		}
	})
}
