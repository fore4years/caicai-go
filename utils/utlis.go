package utils

// Ternary 三目运算
func Ternary[T any](condition bool, a, b T) T {
	if condition {
		return a
	}
	return b
}
