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

func Contains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func ToLower(slice []string) []string {
	result := make([]string, len(slice))
	for i, value := range slice {
		result[i] = strings.ToLower(value)
	}
	return result
}
