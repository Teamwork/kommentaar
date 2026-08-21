package openapi2

import (
	"bytes"
	"testing"

	"github.com/teamwork/kommentaar/docparse"
)

func TestExample(t *testing.T) {
	prog := docparse.NewProgram(false)
	prog.Config.Title = "Test Example"
	prog.Config.Version = "v1"
	prog.Config.Packages = []string{"../example/..."}
	prog.Config.Output = WriteYAML

	w := bytes.NewBufferString("")
	err := docparse.FindComments(w, prog)
	if err != nil {
		t.Fatal(err)
	}

	if len(w.String()) < 500 {
		t.Errorf("short output?")
	}
}

func TestDegradeComposition(t *testing.T) {
	cases := []struct {
		name            string
		in              docparse.Schema
		wantDescription string
		wantType        string
	}{
		{
			name: "oneOf becomes prose",
			in: docparse.Schema{Type: "any", OneOf: []*docparse.Schema{
				{Type: "string"},
				{Type: "number"},
				{Type: "array", Items: &docparse.Schema{Type: "string"}},
			}},
			wantDescription: "One of: string, number, string[].",
			wantType:        "any",
		},
		{
			name: "anyOf becomes prose",
			in: docparse.Schema{AnyOf: []*docparse.Schema{
				{Type: "string"}, {Type: "boolean"},
			}},
			wantDescription: "Any of: string, boolean.",
		},
		{
			name: "keeps the existing description",
			in: docparse.Schema{
				Description: "The stored value",
				OneOf:       []*docparse.Schema{{Type: "string"}, {Type: "number"}},
			},
			wantDescription: "The stored value. One of: string, number.",
		},
		{
			name: "does not double the full stop",
			in: docparse.Schema{
				Description: "The stored value.",
				OneOf:       []*docparse.Schema{{Type: "string"}, {Type: "number"}},
			},
			wantDescription: "The stored value. One of: string, number.",
		},
		{
			name:            "leaves a schema with no alternatives alone",
			in:              docparse.Schema{Type: "string", Description: "Plain"},
			wantDescription: "Plain",
			wantType:        "string",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.in
			degradeComposition(&s)

			if s.OneOf != nil || s.AnyOf != nil {
				t.Error("oneOf/anyOf must not reach the Swagger 2.0 output")
			}
			if s.Description != tc.wantDescription {
				t.Errorf("description: got %q, want %q", s.Description, tc.wantDescription)
			}
			if s.Type != tc.wantType {
				t.Errorf("type: got %q, want %q", s.Type, tc.wantType)
			}
		})
	}
}

func TestSwaggerNullable(t *testing.T) {
	yes := true
	s := &docparse.Schema{
		Properties: map[string]*docparse.Schema{
			"middleName": {Nullable: &yes},
			"lastName":   {},
			"nested": {Properties: map[string]*docparse.Schema{
				"nickname": {Nullable: &yes},
			}},
			"list": {Items: &docparse.Schema{Nullable: &yes}},
		},
	}

	swaggerNullable(s)

	check := func(where string, got *docparse.Schema) {
		if got.Nullable != nil {
			t.Errorf("%s: nullable survived; Swagger 2.0 spells it x-nullable", where)
		}
		if got.XNullable == nil || !*got.XNullable {
			t.Errorf("%s: XNullable = %v, want true", where, got.XNullable)
		}
	}
	check("middleName", s.Properties["middleName"])
	check("nested.nickname", s.Properties["nested"].Properties["nickname"])
	check("list.items", s.Properties["list"].Items)

	if s.Properties["lastName"].XNullable != nil {
		t.Error("a field without the tag must stay unannotated")
	}
}
