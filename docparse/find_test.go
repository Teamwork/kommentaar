package docparse

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/teamwork/test"
)

func TestExampleDir(t *testing.T) {
	prog := NewProgram(false)
	prog.Config.Paths = []string{"../example"}
	prog.Config.Output = func(_ io.Writer, p *Program) error {
		if len(p.Endpoints) < 2 {
			t.Errorf("len(p.Endpoints) == %v", len(p.Endpoints))
		}
		if len(p.References) < 2 {
			t.Errorf("len(p.References) == %v", len(p.References))
		}

		return nil
	}

	err := FindComments(os.Stdout, prog)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFindType(t *testing.T) {
	t.Run("absolute", func(t *testing.T) {
		ts, path, pkg, err := findType("", "net/http", "Header")
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
		tsCached, pathCached, pkgCached, err := findType("", "net/http", "Header")
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
		ts, path, pkg, err := findType("../example/example.go", "exampleimport", "Foo")
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

	t.Run("error", func(t *testing.T) {
		tests := []struct {
			name                      string
			inFile, inPkgPath, inName string
			wantErr                   string
		}{
			{
				"nodir",
				"../../example.com", "asdasd", "qwewqe",
				"no such file or directory",
			},
			{
				"nopkg",
				"", "asdasd", "qwewqe",
				`cannot find package "asdasd" in any of`,
			},
			{
				"notfound",
				"../example/example.go", "exampleimport", "doesntexist",
				`could not find type "doesntexist" in package "github.com/teamwork/kommentaar/example/exampleimport"`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, _, _, err := findType(tt.inFile, tt.inPkgPath, tt.inName)
				if !test.ErrorContains(err, tt.wantErr) {
					t.Fatalf("\nwant: %v\ngot:  %v", tt.wantErr, err)
				}
			})
		}
	})
}
