package docparse

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestFindComments(t *testing.T) {
	InitProgram(false)
	err := FindComments(os.Stdout, []string{"../example"}, func(_ io.Writer, p Program) error {
		if len(p.Endpoints) < 2 {
			t.Errorf("len(p.Endpoints) == %v", len(p.Endpoints))
		}
		if len(p.References) < 2 {
			t.Errorf("len(p.References) == %v", len(p.References))
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestFindType(t *testing.T) {
	t.Run("absolute", func(t *testing.T) {
		ts, path, pkg, err := FindType("", "net/http", "Header")
		if err != nil {
			t.Fatal(err)
		}
		if ts == nil {
			t.Fatal("t is nil")
		}
		if ts.Name.Name != "Header" {
			t.Fatalf("ts.Name.Name == %v", ts.Name.Name)
		}
		if pkg != "net/http" {
			t.Fatalf("pkg == %v", pkg)
		}
		if path != filepath.Join(runtime.GOROOT(), "src", "net", "http", "header.go") {
			t.Fatalf("path == %v", path)
		}

		p, ok := declsCache["net/http"]
		if !ok {
			t.Fatal("not stored in cache?")
		}

		if len(p) < 100 {
			t.Errorf("len(p) == %v", len(p))
		}

		// Make sure it works from cache as well.
		tsCached, pathCached, pkgCached, err := FindType("", "net/http", "Header")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ts, tsCached) {
			t.Error("not equal from cache?")
		}
		if pkg != pkgCached {
			t.Fatalf("pkgCache == %v", pkgCached)
		}
		if path != pathCached {
			t.Fatalf("path == %v", pathCached)
		}
	})

	t.Run("relative", func(t *testing.T) {
		ts, path, pkg, err := FindType("../example/example.go", "exampleimport", "Foo")
		if err != nil {
			t.Fatal(err)
		}
		if ts.Name.Name != "Foo" {
			t.Fatalf("ts.Name.Name == %v", ts.Name.Name)
		}
		if pkg != "github.com/teamwork/kommentaar/example/exampleimport" {
			t.Fatalf("pkg == %v", pkg)
		}
		p, _ := filepath.Abs("./../example/exampleimport/exampleimport.go")
		if path != p {
			t.Fatalf("path == %v", path)
		}
	})
}
