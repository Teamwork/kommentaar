package main // import "github.com/teamwork/kommentaar"

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kr/pretty"
	"github.com/teamwork/kommentaar/docparse"
	"github.com/teamwork/kommentaar/finder"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: kommentaar [pkg pkg...]\n\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()
	paths := flag.Args()

	if len(paths) == 0 {
		paths = []string{"."}
	}

	//err := finder.Find(paths, docparse.Parse, openapi3.Write)
	err := finder.Find(paths, docparse.Parse, dumpOut)
	//err := finder.Find(paths, dumpDoc, dumpOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
}

func dumpDoc(comment, pkg string) (*docparse.Endpoint, error) {
	//fmt.Println(pkg)
	return nil, nil
}

func dumpOut(_ io.Writer, endpoints []*docparse.Endpoint) error {
	_, err := pretty.Print(endpoints)
	if err != nil {
		return err
	}
	fmt.Println("")
	fmt.Println("")
	_, err = pretty.Print(docparse.Refs)
	fmt.Println("")
	return err
}
