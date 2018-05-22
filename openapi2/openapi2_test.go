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
	prog.Config.Paths = []string{"../example/..."}
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
