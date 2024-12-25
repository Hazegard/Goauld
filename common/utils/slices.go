package utils

func Unique[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := []T{}

	for _, value := range slice {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}

	return result
}
