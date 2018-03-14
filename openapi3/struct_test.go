package openapi3

import (
	"go/ast"
	"go/build"
	"testing"

	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/test/diff"
)

func TestFieldToProperty(t *testing.T) {
	docparse.InitProgram(false)

	want := map[string]*SchemaProperty{
		"str":    {Type: "string", Description: "Documented str field.\nNewline."},
		"byt":    {Type: "string"},
		"r":      {Type: "string"},
		"b":      {Type: "boolean", Description: "Inline docs."},
		"fl":     {Type: "number"},
		"err":    {Type: "string"},
		"strP":   {Type: "string"},
		"slice":  {Type: "array", Items: Schema{Type: "string"}},
		"sliceP": {Type: "array", Items: Schema{Type: "string"}},
		"cstr":   {Type: "string"},
		"cstrP":  {Type: "string"},
		"bar": {Type: "object", Properties: map[string]Schema{
			"str": {Type: "string"},
			"num": {Type: "integer", Description: "uint32 docs!"},
		}},
		"barP": {Type: "object", Properties: map[string]Schema{
			"str": {Type: "string"},
			"num": {Type: "integer", Description: "uint32 docs!"},
		}},
		"pkg": {Type: ""},
		//"pkgSlice":  {Type: "array", Items: Schema{Type: "Address"}},
		//"pkgSliceP": {Type: "array", Items: Schema{Type: "Address"}},

	}

	build.Default.GOPATH = "./testdata"
	ts, _, err := docparse.FindType("", "a", "foo")
	if err != nil {
		t.Fatalf("could not parse file: %v", err)
	}

	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		t.Fatal("not a struct?!")
	}

	for _, f := range st.Fields.List {
		out, err := fieldToProperty(docparse.Reference{
			Package: "a",
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
				//if !reflect.DeepEqual(out, w) {
				//	t.Errorf("\nwant: %#v\nout:  %#v", w, out)
				//}
			})
		}
	}

}
