package utils

import "strings"

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

func ToLower(slice []string) []string {
	result := make([]string, len(slice))
	for i, value := range slice {
		result[i] = strings.ToLower(value)
	}
	return result
}
