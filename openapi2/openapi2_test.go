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
