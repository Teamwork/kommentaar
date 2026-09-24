package a

// badNested has an unexported field with a tag, so GetReference returns an
// error for it.
type badNested struct {
	x string `json:"x"`
}

// hasBadNested holds a field of type badNested.
type hasBadNested struct {
	Bad badNested
}

// invalidRefs references badNested as a plain field, a slice element and a
// map value.
type invalidRefs struct {
	bad    badNested
	bads   []badNested
	badMap map[string]badNested
}

// unresolved holds a field of a package that package a does not import.
type unresolved struct {
	X nopkg.Type
}
