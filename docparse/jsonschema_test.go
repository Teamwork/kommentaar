package docparse

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"testing"

	"github.com/teamwork/test"
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
		"dotted":    {Type: "array", Items: &Schema{Reference: "m.Item"}},
		"dottedSlice": {Type: "array", Items: &Schema{
			Type: "array", Items: &Schema{Reference: "m.Item"}}},
		"namedSlice": {Type: "array", Items: &Schema{
			Type: "array", Items: &Schema{Reference: "a.bar"}}},
		"aliasNamedSlice": {Type: "array", Items: &Schema{
			Type: "array", Items: &Schema{Reference: "c.Nested"}}},
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
			assertReferencesDefined(t, prog, out)

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
					"a.ignored":    "string",
					"a.bar":        "string",
				},
				want: map[string]*Schema{
					"b":            {Type: "string"},
					"bSlice":       {Type: "array", Items: &Schema{Type: "string"}},
					"pkg":          {Type: "string"},
					"pkgSlice":     {Type: "array", Items: &Schema{Type: "string"}},
					"ignoredSlice": {Type: "array", Items: &Schema{Type: "string"}},
				},
			},
			{
				name: "fully-qualified key",
				mapTypes: map[string]string{
					"net/mail.Address": "string",
					"a.ignored":        "string",
					"a.bar":            "string",
				},
				want: map[string]*Schema{
					"b":            {Type: "string"},
					"bSlice":       {Type: "array", Items: &Schema{Type: "string"}},
					"pkg":          {Type: "string"},
					"pkgSlice":     {Type: "array", Items: &Schema{Type: "string"}},
					"ignoredSlice": {Type: "array", Items: &Schema{Type: "string"}},
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
				Context: "req",
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

func TestParseCompositionTypes(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    []*Schema
		wantErr bool
	}{
		{"scalars", "string number", []*Schema{{Type: "string"}, {Type: "number"}}, false},
		{
			"array alternative",
			"string []string",
			[]*Schema{{Type: "string"}, {Type: "array", Items: &Schema{Type: "string"}}},
			false,
		},
		{
			"object and boolean",
			"object boolean",
			[]*Schema{{Type: "object"}, {Type: "boolean"}},
			false,
		},
		{
			"extra whitespace and newlines",
			"string\n  number",
			[]*Schema{{Type: "string"}, {Type: "number"}},
			false,
		},
		{"unknown type", "string bogus", nil, true},
		{"unknown array type", "string []bogus", nil, true},
		{"single type is not a union", "string", nil, true},
		{"empty", "", nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseCompositionTypes(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d := diff.Diff(tc.want, got); d != "" {
				t.Error(d)
			}
		})
	}
}

func TestSetTagsComposition(t *testing.T) {
	t.Run("oneof reaches the schema and marshals", func(t *testing.T) {
		var p Schema
		if err := setTags("value", "in.go", &p, []string{"oneof: string number []string"}); err != nil {
			t.Fatalf("setTags: %v", err)
		}
		if len(p.OneOf) != 3 {
			t.Fatalf("got %d alternatives, want 3", len(p.OneOf))
		}

		j, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		want := `{"oneOf":[{"type":"string"},{"type":"number"},` +
			`{"type":"array","items":{"type":"string"}}]}`
		if string(j) != want {
			t.Errorf("got %s, want %s", j, want)
		}
	})

	t.Run("anyof reaches the schema", func(t *testing.T) {
		var p Schema
		if err := setTags("value", "in.go", &p, []string{"anyof: string boolean"}); err != nil {
			t.Fatalf("setTags: %v", err)
		}
		if len(p.AnyOf) != 2 {
			t.Fatalf("got %d alternatives, want 2", len(p.AnyOf))
		}
	})

	t.Run("a bad type list is an error, not a silent pass", func(t *testing.T) {
		var p Schema
		if err := setTags("value", "in.go", &p, []string{"oneof: string bogus"}); err == nil {
			t.Error("expected an error for an unknown type")
		}
	})
}
func TestSetTagsNullable(t *testing.T) {
	// A pointer field opts into the nullable keyword with a doc tag, so only
	// the fields that mean it get annotated.
	var p Schema
	if err := setTags("middleName", "in.go", &p, []string{paramNullable}); err != nil {
		t.Fatalf("setTags: %v", err)
	}
	if p.Nullable == nil || !*p.Nullable {
		t.Errorf("Nullable = %v, want true", p.Nullable)
	}
	if p.XNullable != nil {
		t.Error("docparse holds the OpenAPI 3 spelling; openapi2 moves it")
	}

	// Without the tag nothing is emitted, so existing specs do not change.
	var untagged Schema
	if err := setTags("lastName", "in.go", &untagged, []string{paramReadOnly}); err != nil {
		t.Fatalf("setTags: %v", err)
	}
	if untagged.Nullable != nil {
		t.Errorf("Nullable = %v, want nil", untagged.Nullable)
	}
}

func TestResolveMap(t *testing.T) {
	want := map[string]*Schema{
		"prim":   {Type: "object", AdditionalProperties: &Schema{Type: "integer"}},
		"primP":  {Type: "object", AdditionalProperties: &Schema{Type: "integer"}},
		"anyVal": {Type: "object"},
		"strct":  {Type: "object", AdditionalProperties: &Schema{Reference: "a.bar"}},
		"strctP": {Type: "object", AdditionalProperties: &Schema{Reference: "a.bar"}},
		"pkg":    {Type: "object", AdditionalProperties: &Schema{Reference: "mail.Address"}},
		"slice":  {Type: "object", AdditionalProperties: &Schema{Type: "array", Items: &Schema{Reference: "a.bar"}}},
		"nested": {Type: "object", AdditionalProperties: &Schema{
			Type: "object", AdditionalProperties: &Schema{Reference: "a.bar"}}},
		"sliceOfMap": {Type: "object", AdditionalProperties: &Schema{
			Type: "array", Items: &Schema{Reference: "mail.Address"}}},
		"aliasPkg": {Type: "object", AdditionalProperties: &Schema{Reference: "c.Nested"}},
		"aliasPkgSlice": {Type: "object", AdditionalProperties: &Schema{
			Type: "array", Items: &Schema{Reference: "c.Nested"}}},
		"namedSlice": {Type: "object", AdditionalProperties: &Schema{
			Type: "array", Items: &Schema{Reference: "a.bar"}}},
		"aliasNamedSlice": {Type: "object", AdditionalProperties: &Schema{
			Type: "array", Items: &Schema{Reference: "c.Nested"}}},
		"dotted": {Type: "object", AdditionalProperties: &Schema{Reference: "m.Item"}},
		"dottedSlice": {Type: "object", AdditionalProperties: &Schema{
			Type: "array", Items: &Schema{Reference: "m.Item"}}},
		"selfRef": {Type: "object", AdditionalProperties: &Schema{Reference: "c.Tree"}},
	}

	build.Default.GOPATH = "./testdata"
	ts, _, _, err := findType("./testdata/src/a/a.go", "a", "maps")
	if err != nil {
		t.Fatalf("could not parse file: %v", err)
	}

	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		t.Fatal("not a struct?!")
	}

	for _, f := range st.Fields.List {
		name := f.Names[0].Name
		t.Run(name, func(t *testing.T) {
			typ, ok := f.Type.(*ast.MapType)
			if !ok {
				t.Fatalf("%v is not a map but a %T", name, f.Type)
			}

			w, ok := want[name]
			if !ok {
				t.Fatalf("no test case for %v", name)
			}

			prog := NewProgram(false)
			out := &Schema{}
			ref := Reference{
				Package: "a",
				File:    "./testdata/src/a/a.go",
				Context: "req",
			}
			if err := resolveMap(prog, ref, "a", out, typ, nil); err != nil {
				t.Fatal(err)
			}

			if d := diff.Diff(w, out); d != "" {
				t.Errorf("%v", d)
			}
			assertReferencesDefined(t, prog, out)
		})
	}
}

// TestResolveMapStoredReference makes sure that a second map of a stored type
// takes the stored definition. The type is on a dotted import path, which
// GetReference does not find in prog.References by its full lookup.
func TestResolveMapStoredReference(t *testing.T) {
	build.Default.GOPATH = "./testdata"
	ts, _, _, err := findType("./testdata/src/a/a.go", "a", "maps")
	if err != nil {
		t.Fatalf("could not parse file: %v", err)
	}

	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		t.Fatal("not a struct?!")
	}

	var typ *ast.MapType
	for _, f := range st.Fields.List {
		if f.Names[0].Name == "dotted" {
			typ, _ = f.Type.(*ast.MapType)
		}
	}
	if typ == nil {
		t.Fatal("no dotted map field")
	}

	prog := NewProgram(false)
	for _, ctx := range []string{"req", "resp"} {
		out := &Schema{}
		ref := Reference{Package: "a", File: "./testdata/src/a/a.go", Context: ctx}
		if err := resolveMap(prog, ref, "a", out, typ, nil); err != nil {
			t.Fatalf("%v: %v", ctx, err)
		}
		assertReferencesDefined(t, prog, out)
	}

	if got := prog.References["m.Item"].Context; got != "req" {
		t.Errorf("m.Item context = %q, want %q", got, "req")
	}
}

// assertReferencesDefined reports every $ref in s that has no definition in
// prog.References. A reference that nothing defines gives an unusable
// document.
func assertReferencesDefined(t *testing.T, prog *Program, s *Schema) {
	t.Helper()
	if s == nil {
		return
	}
	if s.Reference != "" {
		if _, ok := prog.References[s.Reference]; !ok {
			t.Errorf("no definition for reference %q; defined: %v",
				s.Reference, referenceNames(prog))
		}
	}
	assertReferencesDefined(t, prog, s.AdditionalProperties)
	assertReferencesDefined(t, prog, s.Items)
}

func fieldNames(fields []Param) []string {
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}

func referenceNames(prog *Program) []string {
	names := make([]string, 0, len(prog.References))
	for name := range prog.References {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TestResolveMapPackageCollision makes sure that two types of the same name
// from two packages that share a base name get one reference each. Both
// resolve through the same Program, which is where GetReference renames the
// second definition.
func TestResolveMapPackageCollision(t *testing.T) {
	build.Default.GOPATH = "./testdata"
	ts, _, _, err := findType("./testdata/src/a/a.go", "a", "mapsCollide")
	if err != nil {
		t.Fatalf("could not parse file: %v", err)
	}

	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		t.Fatal("not a struct?!")
	}

	want := map[string]struct{ ref, pkg string }{
		"first":  {"c.Nested", "c"},
		"second": {"c.Nested2", "d/c"},
	}

	prog := NewProgram(false)
	for _, f := range st.Fields.List {
		name := f.Names[0].Name
		typ, ok := f.Type.(*ast.MapType)
		if !ok {
			t.Fatalf("%v is not a map but a %T", name, f.Type)
		}

		out := &Schema{}
		err := resolveMap(prog, Reference{
			Package: "a",
			File:    "./testdata/src/a/a.go",
			Context: "req",
		}, "a", out, typ, nil)
		if err != nil {
			t.Fatalf("%v: %v", name, err)
		}
		assertReferencesDefined(t, prog, out)
		if out.AdditionalProperties == nil {
			t.Fatalf("%v: no additionalProperties", name)
		}

		got := out.AdditionalProperties.Reference
		if got != want[name].ref {
			t.Errorf("%v: reference = %q, want %q", name, got, want[name].ref)
		}
		if pkg := prog.References[got].Package; pkg != want[name].pkg {
			t.Errorf("%v: %q is from package %q, want %q", name, got, pkg, want[name].pkg)
		}
	}
}

// TestFieldToPropertyPackageCollision makes sure a plain field, a pointer
// field and a slice element each get their own $ref when their type's base
// name collides with a type from another package.
func TestFieldToPropertyPackageCollision(t *testing.T) {
	orig := build.Default.GOPATH
	build.Default.GOPATH = "./testdata"
	defer func() { build.Default.GOPATH = orig }()
	ts, _, _, err := findType("./testdata/src/a/a.go", "a", "collide")
	if err != nil {
		t.Fatalf("could not parse file: %v", err)
	}

	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		t.Fatal("not a struct?!")
	}

	want := map[string]struct {
		ref, pkg string
		fields   []string
	}{
		"first":       {"report.Nested", "repa/report", []string{"Str"}},
		"firstP":      {"report.Nested", "repa/report", []string{"Str"}},
		"firstSlice":  {"report.Nested", "repa/report", []string{"Str"}},
		"second":      {"report.Nested2", "repb/report", []string{"Num"}},
		"secondP":     {"report.Nested2", "repb/report", []string{"Num"}},
		"secondSlice": {"report.Nested2", "repb/report", []string{"Num"}},
	}

	prog := NewProgram(false)
	ref := Reference{Package: "a", File: "./testdata/src/a/a.go", Context: "req"}
	for _, f := range st.Fields.List {
		name := f.Names[0].Name
		out, err := fieldToSchema(prog, name, "json", ref, f, nil)
		if err != nil {
			t.Fatalf("%v: %v", name, err)
		}
		assertReferencesDefined(t, prog, out)

		got := out.Reference
		if out.Items != nil {
			got = out.Items.Reference
		}
		if got != want[name].ref {
			t.Errorf("%v: reference = %q, want %q", name, got, want[name].ref)
		}
		stored := prog.References[got]
		if stored.Package != want[name].pkg {
			t.Errorf("%v: %q is from package %q, want %q", name, got, stored.Package, want[name].pkg)
		}
		if fields := fieldNames(stored.Fields); !reflect.DeepEqual(fields, want[name].fields) {
			t.Errorf("%v: %q fields = %v, want %v", name, got, fields, want[name].fields)
		}
	}
}

// TestGetReferenceEmbedPackageCollision makes sure an embedded struct merges
// the fields of the type that its own package declares, not a type from
// another package that shares its base name.
func TestGetReferenceEmbedPackageCollision(t *testing.T) {
	orig := build.Default.GOPATH
	build.Default.GOPATH = "./testdata"
	defer func() { build.Default.GOPATH = orig }()
	prog := NewProgram(false)
	prog.Config.StructTag = "json"

	first, err := GetReference(prog, "req", false, "a.collideEmbedFirst", "./testdata/src/a/a.go")
	if err != nil {
		t.Fatalf("collideEmbedFirst: %v", err)
	}
	second, err := GetReference(prog, "resp", false, "a.collideEmbedSecond", "./testdata/src/a/a.go")
	if err != nil {
		t.Fatalf("collideEmbedSecond: %v", err)
	}

	if got, want := fieldNames(first.Fields), []string{"Str"}; !reflect.DeepEqual(got, want) {
		t.Errorf("collideEmbedFirst fields = %v, want %v", got, want)
	}
	if got, want := fieldNames(second.Fields), []string{"Num"}; !reflect.DeepEqual(got, want) {
		t.Errorf("collideEmbedSecond fields = %v, want %v", got, want)
	}

	nested1, ok := prog.References["report.Nested"]
	if !ok || nested1.Package != "repa/report" {
		t.Errorf("report.Nested = %+v, want Package %q", nested1, "repa/report")
	}
	if nested1.Context != "req" {
		t.Errorf("report.Nested context = %q, want %q", nested1.Context, "req")
	}
	nested2, ok := prog.References["report.Nested2"]
	if !ok || nested2.Package != "repb/report" {
		t.Errorf("report.Nested2 = %+v, want Package %q", nested2, "repb/report")
	}
	if nested2.Context != "resp" {
		t.Errorf("report.Nested2 context = %q, want %q", nested2.Context, "resp")
	}
}

// TestInvalidNestedReference makes sure an error from GetReference for a
// field type gets to the caller. A map value is the exception: resolveMap
// gives an open object.
func TestInvalidNestedReference(t *testing.T) {
	orig := build.Default.GOPATH
	build.Default.GOPATH = "./testdata"
	defer func() { build.Default.GOPATH = orig }()

	t.Run("findNested", func(t *testing.T) {
		prog := NewProgram(false)
		prog.Config.StructTag = "json"
		_, err := GetReference(prog, "req", false, "a.hasBadNested", "./testdata/src/a/a.go")
		if !test.ErrorContains(err, "not exported") {
			t.Errorf("err = %v, want %q", err, "not exported")
		}
	})

	ts, _, _, err := findType("./testdata/src/a/a.go", "a", "invalidRefs")
	if err != nil {
		t.Fatalf("could not parse file: %v", err)
	}
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		t.Fatal("not a struct?!")
	}

	wantErr := map[string]string{
		"bad":    "not exported",
		"bads":   "not exported",
		"badMap": "",
	}
	for _, f := range st.Fields.List {
		name := f.Names[0].Name
		t.Run(name, func(t *testing.T) {
			prog := NewProgram(false)
			prog.Config.StructTag = "json"
			out, err := fieldToSchema(prog, name, "json", Reference{
				Package: "a",
				File:    "./testdata/src/a/a.go",
				Context: "req",
			}, f, nil)
			if !test.ErrorContains(err, wantErr[name]) {
				t.Fatalf("err = %v, want %q", err, wantErr[name])
			}
			if err == nil && out.AdditionalProperties != nil {
				t.Errorf("additionalProperties = %+v, want nil", out.AdditionalProperties)
			}
		})
	}
}
