//go:build ignore

package a

// ignored is in a file that the build excludes, so findType does not find it.
type ignored string
