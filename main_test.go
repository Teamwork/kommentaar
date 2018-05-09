package main

import (
	"bytes"
	"go/build"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/kommentaar/openapi3"
	"github.com/teamwork/test"
	"github.com/teamwork/test/diff"
)

func TestMain(t *testing.T) {
	os.Args = []string{"", "-config", "config.example", "./example/..."}
	main()
}

func TestOpenAPI3(t *testing.T) {
	tests, err := ioutil.ReadDir("./testdata/src")
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.Name(), func(t *testing.T) {
			path := "./testdata/src/" + tt.Name()

			want, err := ioutil.ReadFile(path + "/want.yaml")
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("could not read output: %v", err)
			}
			want = append(bytes.TrimSpace(want), '\n')

			wantErr, err := ioutil.ReadFile(path + "/wantErr")
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("could not read wantErr: %v", err)
			}
			wantErr = bytes.TrimSpace(wantErr)

			build.Default.GOPATH = "./testdata"

			prog := docparse.NewProgram(false)
			prog.Config.Paths = []string{"./testdata/src/" + tt.Name()}
			prog.Config.Output = openapi3.WriteYAML

			outBuf := bytes.NewBuffer(nil)
			err = docparse.FindComments(outBuf, prog)
			if !test.ErrorContains(err, string(wantErr)) {
				t.Fatalf("wrong error\nout:  %v\nwant: %v", err.Error(), string(wantErr))
			}
			out := strings.TrimSpace(outBuf.String()) + "\n"

			d := diff.TextDiff(string(want), out)
			if d != "" {
				t.Fatalf("wrong output\n%v", d)
			}
		})
	}
}
