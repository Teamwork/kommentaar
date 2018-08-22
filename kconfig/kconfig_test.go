package kconfig

import (
	"testing"

	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/test"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
	}{
		{"example", test.Read(t, "../config.example")},
		{"default-response", []byte(test.NormalizeIndent(`
			default-response 400: $ref: github.com/teamwork/kommentaar/docparse.Param
			default-response 404 (application/json): $ref: net/mail.Address
		`))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, clean := test.TempFile(t, string(tt.in))
			defer clean()

			prog := docparse.NewProgram(false)

			err := Load(prog, f)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
		})
	}
}
