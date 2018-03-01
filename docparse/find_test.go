package docparse

import (
	"io"
	"reflect"
	"testing"
)

func TestFindComments(t *testing.T) {
	InitProgram()
	err := FindComments([]string{"../example"}, func(_ io.Writer, p Program) error {
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
	ts, err := FindType("net/http", "Header")
	if err != nil {
		t.Fatal(err)
	}
	if ts == nil {
		t.Fatal("t is nil")
	}
	if ts.Name.Name != "Header" {
		t.Fatalf("ts.Name.Name == %v", ts.Name.Name)
	}

	p, ok := declsCache["net/http"]
	if !ok {
		t.Fatal("not stored in cache?")
	}

	if len(p) < 100 {
		t.Errorf("len(p) == %v", len(p))
	}

	// Make sure it works from cache as well.
	ts3, err := FindType("net/http", "Header")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ts, ts3) {
		t.Error("not equal from cache?")
	}
}
