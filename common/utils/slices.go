//nolint:revive
package utils

import "strings"

// Unique removes all duplicates in a slice.
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	var result []T

	for _, value := range slice {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}

	return result
}

// Contains returns whether a value is contained in a slice.
func Contains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}

	return false
}

// ToLower returns a slice of lowercase string.
func ToLower(slice []string) []string {
	result := make([]string, len(slice))
	for i, value := range slice {
		result[i] = strings.ToLower(value)
	}

	return result
}
