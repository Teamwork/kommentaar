package generics_cross_pkg_third

import (
	enumpkg "generics-cross-pkg-third/enumpkg"
	genpkg "generics-cross-pkg-third/genpkg"
)

type reqRef struct {
	// Metric documents a generic with a type arg from a third package.
	Metric genpkg.Wrapper[enumpkg.MetricType]
}

// POST /path
//
// Request body: reqRef
// Response 200: {empty}
