package docparse

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/teamwork/test"
	"github.com/teamwork/test/diff"
)

func TestExampleDir(t *testing.T) {
	prog := NewProgram(false)
	prog.Config.Packages = []string{"../example"}
	prog.Config.StructTag = "json"
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

//func TestGetReference(t *testing.T) {
//	prog.Config.Title = "x"
//	prog.Config.Version = "x"
//	prog := NewProgram(false)
//	prog.Config.StructTag = "json"
//
//	//func GetReference(prog *Program, context string, isEmbed bool, lookup, filePath string) (*Reference, error) {
//}

func TestGetReference(t *testing.T) {
	tests := []struct {
		in      string
		wantErr string
		want    *Reference
	}{
		{"testObject", "", &Reference{
			Name:    "testObject",
			Package: "github.com/teamwork/kommentaar/docparse",
			File:    "", // TODO
			Lookup:  "docparse.testObject",
			Context: "req",
			Info:    "testObject general documentation.",
			Fields: []Param{
				{Name: "ID"},
				{Name: "Foo"},
				{Name: "Bar"},
			},
			Schema: &Schema{
				Title:       "testObject",
				Description: "testObject general documentation.",
				Type:        "object",
				Required:    []string{"ID"},
				Properties: map[string]*Schema{
					"ID":  {Type: "integer", Description: "ID documentation."},
					"Foo": {Type: "string", Description: "Foo is a really cool foo-thing!\nSuch foo!"},
					"Bar": {Type: "array", Items: &Schema{Type: "string"}},
				},
			},
		}},
		{"net/mail.Address", "", &Reference{
			Name:    "Address",
			Package: "net/mail",
			File:    "", // TODO
			Lookup:  "mail.Address",
			Context: "req",
			Info: "Address represents a single mail address.\n" +
				"An address such as \"Barry Gibbs <bg@example.com>\" is represented\n" +
				`as Address{Name: "Barry Gibbs", Address: "bg@example.com"}.`,
			Fields: []Param{
				{Name: "Name"},
				{Name: "Address"},
			},
			Schema: &Schema{
				Title: "Address",
				Description: "Address represents a single mail address.\n" +
					"An address such as \"Barry Gibbs <bg@example.com>\" is represented\n" +
					"as Address{Name: \"Barry Gibbs\", Address: \"bg@example.com\"}.",
				Type: "object",
				Properties: map[string]*Schema{
					"Address": {Type: "string", Description: "user@domain"},
					"Name":    {Type: "string", Description: "Proper name; may be empty."},
				},
			},
		}},

		{"UnknownObject", "could not find", nil},
		{"net/http.Header", "not a struct", nil},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.in), func(t *testing.T) {
			prog := NewProgram(true)
			out, err := GetReference(prog, "req", false, tt.in, ".")
			if !test.ErrorContains(err, tt.wantErr) {
				t.Fatalf("wrong err\nout:  %v\nwant: %v\n", err, tt.wantErr)
			}

			if out != nil {
				out.File = "" // TODO: test this as well.
			}
			if out != nil && out.Fields != nil {
				for i := range out.Fields {
					out.Fields[i].KindField = nil
				}
			}

			if !reflect.DeepEqual(tt.want, out) {
				t.Errorf("\n%v", diff.Diff(tt.want, out))
			}

			if tt.wantErr == "" {
				testCache = true
				t.Run("cache", func(t *testing.T) {
					out, err := GetReference(prog, "req", false, tt.in, ".")
					if err != nil {
						t.Fatal(err)
					}

					if out != nil {
						out.File = "" // TODO: test this as well.
					}
					if out != nil && out.Fields != nil {
						for i := range out.Fields {
							out.Fields[i].KindField = nil
						}
					}
					if !reflect.DeepEqual(tt.want, out) {
						t.Errorf("\n%v", diff.Diff(tt.want, out))
					}

					//for k := range prog.References {
					//	fmt.Println("CACHE", k)
					//}
				})
			}
		})
	}
}
