package main

import (
	"os"
	"testing"
)

func TestMain(t *testing.T) {
	os.Args = []string{"", "-config", "config.example", "./example/..."}
	main()
}
