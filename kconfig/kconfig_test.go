package kconfig

import (
	"testing"

	"github.com/teamwork/kommentaar/docparse"
)

func TestLoadExample(t *testing.T) {
	prog := docparse.NewProgram(false)
	err := Load(prog, "../config.example")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
}
