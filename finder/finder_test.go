package finder

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/teamwork/test"
)

// This also tests ResolvePackage() and ResolveWildcard().
func TestExpand(t *testing.T) {
	cases := []struct {
		in      []string
		want    []string
		wantErr string
	}{
		{
			[]string{"fmt"},
			[]string{"fmt"},
			"",
		},
		{
			[]string{"fmt", "net/http"},
			[]string{"fmt", "net/http"},
			"",
		},
		{
			[]string{"net/..."},
			[]string{"net", "net/http", "net/http/cgi", "net/http/cookiejar",
				"net/http/fcgi", "net/http/httptest", "net/http/httptrace",
				"net/http/httputil", "net/http/internal", "net/http/pprof",
				"net/internal/socktest", "net/mail", "net/rpc", "net/rpc/jsonrpc",
				"net/smtp", "net/textproto", "net/url",
			},
			"",
		},
		{
			[]string{"github.com/teamwork/kommentaar"},
			[]string{"github.com/teamwork/kommentaar"},
			"",
		},
		{
			[]string{"."},
			[]string{"github.com/teamwork/kommentaar/finder"},
			"",
		},
		{
			[]string{".."},
			[]string{"github.com/teamwork/kommentaar"},
			"",
		},
		{
			[]string{"../..."},
			[]string{"github.com/teamwork/kommentaar",
				"github.com/teamwork/kommentaar/docparse",
				"github.com/teamwork/kommentaar/example",
				"github.com/teamwork/kommentaar/finder",
				"github.com/teamwork/kommentaar/openapi3",
			},
			"",
		},

		// Errors
		{
			[]string{""},
			nil,
			"cannot resolve empty string",
		},
		{
			[]string{"this/will/never/exist"},
			nil,
			`cannot find package "this/will/never/exist"`,
		},
		{
			[]string{"this/will/never/exist/..."},
			nil,
			`cannot find package "this/will/never/exist"`,
		},
		{
			[]string{"./doesnt/exist"},
			nil,
			"cannot find package",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			out, err := Expand(tc.in)
			if !test.ErrorContains(err, tc.wantErr) {
				t.Fatal(err)
			}

			sort.Strings(tc.want)
			var outPkgs []string
			for _, p := range out {
				outPkgs = append(outPkgs, p.ImportPath)
			}

			if !reflect.DeepEqual(tc.want, outPkgs) {
				t.Errorf("\nout:  %#v\nwant: %#v\n", outPkgs, tc.want)
			}
		})
	}
}
