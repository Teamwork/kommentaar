package html

import (
	"bytes"
	"testing"

	"github.com/teamwork/kommentaar/docparse"
)

func TestHTML(t *testing.T) {
	prog := docparse.NewProgram(false)
	prog.Config.Packages = []string{"../example/..."}
	prog.Config.Output = WriteHTML

	w := bytes.NewBufferString("")
	err := docparse.FindComments(w, prog)
	if err != nil {
		t.Fatal(err)
	}

	if len(w.String()) < 500 {
		t.Errorf("short output?")
	}
}
