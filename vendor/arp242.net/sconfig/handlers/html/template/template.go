// Copyright © 2016-2018 Martin Tournoij
// See the bottom of this file for the full copyright.

// Package template contains handlers for parsing values with the html/template package.
package template // import "arp242.net/sconfig/handlers/html/template"

import (
	"html/template"
	"strings"

	"arp242.net/sconfig"
)

func init() {
	sconfig.RegisterType("template.HTML", handleHTML)
}

func handleHTML(v []string) (interface{}, error) {
	return template.HTML(strings.Join(v, " ")), nil
}

// The MIT License (MIT)
//
// Copyright © 2016-2018 Martin Tournoij
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// The software is provided "as is", without warranty of any kind, express or
// implied, including but not limited to the warranties of merchantability,
// fitness for a particular purpose and noninfringement. In no event shall the
// authors or copyright holders be liable for any claim, damages or other
// liability, whether in an action of contract, tort or otherwise, arising
// from, out of or in connection with the software or the use or other dealings
// in the software.
