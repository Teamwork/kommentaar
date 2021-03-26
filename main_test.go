package main

import (
	"bytes"
	"flag"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/kommentaar/kconfig"
	"github.com/teamwork/kommentaar/openapi2"
	"github.com/teamwork/test"
	"github.com/teamwork/test/diff"
)

// Just basic sanity test to make sure it doesn't error out or something.
func TestStart(t *testing.T) {
	os.Args = []string{"", "-config", "config.example", "./example/..."}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	stdout = bytes.NewBufferString("")
	_, err := start()
	if err != nil {
		t.Error(err)
	}
}

func TestOpenAPI2(t *testing.T) {
	tests, err := ioutil.ReadDir("./testdata/openapi2/src")
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		if tt.Name() != "package-alias" {
			continue
		}
		t.Run(tt.Name(), func(t *testing.T) {
			path := "./testdata/openapi2/src/" + tt.Name()

			want, err := ioutil.ReadFile(path + "/want.yaml")
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("could not read output: %v", err)
			}
			want = append(bytes.TrimSpace(want), '\n')

			wantJSON, err := ioutil.ReadFile(path + "/want.json")
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("could not read output: %v", err)
			}
			wantJSON = append(bytes.TrimSpace(wantJSON), '\n')

			wantErr, err := ioutil.ReadFile(path + "/wantErr")
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("could not read wantErr: %v", err)
			}
			wantErr = bytes.TrimSpace(wantErr)

			wd, _ := os.Getwd()
			build.Default.GOPATH = filepath.Join(wd, "/testdata/openapi2")

			prog := docparse.NewProgram(false)
			prog.Config.Title = "x"
			prog.Config.Version = "x"
			prog.Config.Packages = []string{"./testdata/openapi2/src/" + tt.Name()}
			prog.Config.Output = openapi2.WriteYAML
			prog.Config.StructTag = "json"

			// Allow test to override config
			testConfig := path + "/test.conf"
			if _, err := os.Stat(testConfig); err == nil {
				if err := kconfig.Load(prog, testConfig); err != nil {
					t.Fatalf("test.conf: %v", err)
				}
			}

			outBuf := bytes.NewBuffer(nil)
			err = docparse.FindComments(outBuf, prog)
			if !test.ErrorContains(err, string(wantErr)) {
				t.Fatalf("wrong error\nout:  %v\nwant: %v", err, string(wantErr))
			}
			out := strings.TrimSpace(outBuf.String()) + "\n"

			d := diff.TextDiff(string(want), out)
			if d != "" {
				t.Errorf("incorrect output: \n%s", out)
				t.Fatalf("diff\n%v", d)
			}

			if len(wantJSON) > 1 {
				prog.Config.Output = openapi2.WriteJSONIndent
				prog.Endpoints = nil
				prog.References = make(map[string]docparse.Reference)
				outBuf := bytes.NewBuffer(nil)
				err = docparse.FindComments(outBuf, prog)
				if err != nil {
					t.Fatalf("JSON error: %v", err)
				}
				out := strings.TrimSpace(outBuf.String()) + "\n"

				d := diff.TextDiff(string(wantJSON), out)
				if d != "" {
					t.Fatalf("wrong JSON output\n%v", d)
				}
			}
		})
	}
}
