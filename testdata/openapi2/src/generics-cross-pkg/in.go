package generics_cross_pkg

import genpkg "generics-cross-pkg/genpkg"

type reqRef struct {
	// Foo documents a cross-package generic with single type param.
	Foo genpkg.MyGeneric[string]
	// Bar documents a cross-package generic with two type params.
	Bar genpkg.MyGenericMulti[string, genpkg.CustomType]
}

// POST /path
//
// Request body: reqRef
// Response 200: {empty}
