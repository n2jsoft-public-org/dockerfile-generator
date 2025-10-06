// Package util provides small generic helper functions (Where, SelectMany) used internally.
// revive:disable:var-naming - 'util' is a conventional short package name accepted here.
package util

// Where filters arr returning elements for which predicate f returns true.
func Where[T any](arr []T, f func(T) bool) (result []T) {
	for _, item := range arr {
		if f(item) {
			result = append(result, item)
		}
	}
	return
}

// SelectMany projects each element to a slice and flattens the results.
func SelectMany[T, U any](arr []T, f func(T) []U) (result []U) {
	for _, item := range arr {
		result = append(result, f(item)...)
	}
	return
}
