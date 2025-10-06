package util

func Where[T any](arr []T, f func(T) bool) (result []T) {
	for _, item := range arr {
		if f(item) {
			result = append(result, item)
		}
	}
	return
}

func SelectMany[T, U any](arr []T, f func(T) []U) (result []U) {
	for _, item := range arr {
		result = append(result, f(item)...)
	}
	return
}
