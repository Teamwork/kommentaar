package sliceutil

// JoinInt see Join
//
// Deprecated: use Join
func JoinInt(ints []int64) string {
	return Join(ints)
}

// UniqInt64 see Unique
//
// Deprecated: use Unique
func UniqInt64(list []int64) []int64 {
	return Unique(list)
}

// UniqString see Unique
//
// Deprecated: use Unique
func UniqString(list []string) []string {
	return Unique(list)
}

// UniqueMergeSlices see MergeUnique
//
// Deprecated: use MergeUnique
func UniqueMergeSlices(s [][]int64) (result []int64) {
	return MergeUnique(s)
}

// InStringSlice see Contains
//
// Deprecated: use Contains
func InStringSlice(list []string, str string) bool {
	return Contains(list, str)
}

// InIntSlice see Contains
//
// Deprecated: use Contains
func InIntSlice(list []int, i int) bool {
	return Contains(list, i)
}

// InInt64Slice see Contains
//
// Deprecated: use Contains
func InInt64Slice(list []int64, i int64) bool {
	return Contains(list, i)
}

// RepeatString see Repeat
//
// Deprecated: use Repeat
func RepeatString(s string, n int) (r []string) {
	return Repeat(s, n)
}

// ChooseString see Choose
//
// Deprecated: use Choose
func ChooseString(l []string) string {
	return Choose(l)
}

// FilterString see Filter
//
// Deprecated: use Filter
func FilterString(list []string, fun func(string) bool) []string {
	return Filter(list, fun)
}

// RemoveString see Remove
//
// Deprecated: use Remove
func RemoveString(list []string, s string) (out []string) {
	return Remove(list, s)
}

// FilterStringEmpty see FilterEmpty
//
// Deprecated: use FilterEmpty
func FilterStringEmpty(e string) bool {
	return FilterEmpty(e)
}

// FilterInt see Filter
//
// Deprecated: use Filter
func FilterInt(list []int64, fun func(int64) bool) []int64 {
	return Filter(list, fun)
}

// StringMap see Map
//
// Deprecated: use Map
func StringMap(list []string, f func(string) string) []string {
	return Map(list, f)
}
