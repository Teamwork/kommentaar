package path

import "description-print-var/otherpkg"

var _ = otherpkg.DocRatelimitHeader

const (
	str       = "value of constant string"
	ratelimit = 500
)

var (
	strs = []string{"value", "of", "str", "slice"}
	ints = []int{1, 2, 3, 4}
	kv   = map[string]string{
		"a": "b",
		"c": "d",
	}
	kv2 = map[string]interface{}{
		"int":   1,
		"bool":  true,
		"false": false,
		"str":   "hello",
		"f":     1.234,
	}
)

// POST /path
//
// $print otherpkg.DocRatelimitHeader
// $print ratelimit
//
// string:
// $print str
//
// []string:
// $print strs
//
// map[string]string:
// $print kv
//
// map[string]interface{}:
// $print kv2
//
// Response 200: {empty}
