package package_name_collision

import (
	sharedA "package-name-collision/sub-a/shared"
	sharedB "package-name-collision/sub-b/shared"
)

// POST /foo create foo
//
// Request body: sharedA.Request
// Response 200: {empty}

// POST /bar create bar
//
// Request body: sharedB.Request
// Response 200: {empty}
