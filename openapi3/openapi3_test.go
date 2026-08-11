package openapi3

import (
	"bytes"
	"strings"
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

	out := w.String()
	if len(out) < 500 {
		t.Errorf("short output?")
	}

	for _, want := range []string{
		"openapi: 3.0.3",
		"components:",
		"schemas:",
		"requestBody:",
		"$ref: '#/components/schemas/",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q", want)
		}
	}

	for _, notWant := range []string{
		"swagger:",
		"definitions:",
		"#/definitions/",
		"in: body",
		"consumes:",
		"produces:",
	} {
		if strings.Contains(out, notWant) {
			t.Errorf("output still contains %q", notWant)
		}
	}
}

func TestStatusText(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{200, "OK"},
		{404, "Not Found"},
		{422, "Unprocessable Entity"},
		{799, "Response 799"},
	}

	for _, tt := range tests {
		if got := statusText(tt.code); got != tt.want {
			t.Errorf("statusText(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestParamSchema(t *testing.T) {
	ref := func(s string) string { return refPrefix + s }

	t.Run("reference wins", func(t *testing.T) {
		got := paramSchema(&docparse.Schema{Reference: "pkg.Type", Type: "object"}, ref)
		if got.Reference != refPrefix+"pkg.Type" {
			t.Errorf("got %q", got.Reference)
		}
		if got.Type != "" {
			t.Errorf("type should be empty, got %q", got.Type)
		}
	})

	t.Run("empty type falls back to string", func(t *testing.T) {
		got := paramSchema(&docparse.Schema{}, ref)
		if got.Type != "string" {
			t.Errorf("got %q, want string", got.Type)
		}
	})

	t.Run("array items keep their reference", func(t *testing.T) {
		got := paramSchema(&docparse.Schema{
			Type:  "array",
			Items: &docparse.Schema{Reference: "pkg.Item"},
		}, ref)
		if got.Items == nil {
			t.Fatal("items is nil")
		}
		if got.Items.Reference != refPrefix+"pkg.Item" {
			t.Errorf("got %q", got.Items.Reference)
		}
	})

	t.Run("scalar fields are copied", func(t *testing.T) {
		got := paramSchema(&docparse.Schema{
			Type:    "integer",
			Format:  "int64",
			Enum:    []string{"1", "2"},
			Default: "1",
			Minimum: 1,
			Maximum: 9,
		}, ref)
		if got.Format != "int64" || got.Default != "1" || got.Minimum != 1 || got.Maximum != 9 {
			t.Errorf("fields not copied: %#v", got)
		}
		if len(got.Enum) != 2 {
			t.Errorf("enum not copied: %#v", got.Enum)
		}
	})
}

func TestMakeID(t *testing.T) {
	if got := makeID("GET", "/foo/bar"); got != "GET_foo_bar" {
		t.Errorf("got %q", got)
	}
}
