package srvhttp

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/teamwork/test/diff"
)

func TestServe(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	args := Args{
		Packages: []string{"../example/..."},
	}

	rr := httptest.NewRecorder()
	YAML(args)(rr, r)
	if len(rr.Body.String()) < 500 {
		t.Error("body too short for YAML?")
	}

	rr = httptest.NewRecorder()
	JSON(args)(rr, r)
	if len(rr.Body.String()) < 500 {
		t.Error("body too short for JSON?")
	}

	rr = httptest.NewRecorder()
	HTML(args)(rr, r)
	if len(rr.Body.String()) < 500 {
		t.Error("body too short for HTML?")
	}
}

func TestFromFile(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	args := Args{
		Packages: []string{"../example/..."},
		NoScan:   true,
		YAMLFile: "../testdata/openapi2/src/blank-line/want.yaml",
	}

	want, err := os.ReadFile("../testdata/openapi2/src/blank-line/want.yaml")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}

	rr := httptest.NewRecorder()
	YAML(args)(rr, r)
	d := diff.TextDiff(string(want), rr.Body.String())
	if d != "" {
		t.Fatalf("wrong output\n%v", d)
	}
}
