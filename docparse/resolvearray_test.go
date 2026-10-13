package docparse

import (
	"go/ast"
	"go/build"
	"strings"
	"testing"

	"github.com/teamwork/test/diff"
)

func TestResolveArrayElements(t *testing.T) {
	cases := map[string]struct {
		want    *Schema
		wantErr string
	}{
		"nestedRef": {want: &Schema{Type: "array", Items: &Schema{
			Type: "array", Items: &Schema{Reference: "elem.item"}}}},
		"nestedEnum": {want: &Schema{Type: "array", Items: &Schema{
			Type: "array", Items: &Schema{Type: "string", Enum: []string{"x", "y"}}}}},
		"nestedChan": {wantErr: "unknown array type: *ast.ChanType"},

		"ch": {wantErr: "unknown array type: *ast.ChanType"},
	}

	build.Default.GOPATH = "./testdata"
	ts, _, _, err := findType("./testdata/src/elem/elem.go", "elem", "elems")
	if err != nil {
		t.Fatalf("could not parse file: %v", err)
	}
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		t.Fatal("not a struct?!")
	}

	ref := Reference{Package: "elem", File: "./testdata/src/elem/elem.go", Context: "req"}
	for _, f := range st.Fields.List {
		name := f.Names[0].Name
		t.Run(name, func(t *testing.T) {
			tc, ok := cases[name]
			if !ok {
				t.Fatalf("no test case for %v", name)
			}

			prog := NewProgram(false)
			out, err := fieldToSchema(prog, name, "json", ref, f, nil)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertReferencesDefined(t, prog, out)
			if d := diff.Diff(tc.want, out); d != "" {
				t.Error(d)
			}
		})
	}
}
