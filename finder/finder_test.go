package finder

import (
	"fmt"
	"testing"
	"time"
)

func TestFindType(t *testing.T) {
	start := time.Now()

	err := FindType("net/http")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(time.Since(start))

	start = time.Now()
	err = FindType("net/http")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(time.Since(start))
}
