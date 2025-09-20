package utils

func Pointer[T any](a T) *T {
	return &a
}
