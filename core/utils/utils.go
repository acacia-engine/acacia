package utils

import (
	"fmt"
	"os"
	"regexp"
)

// This file will contain generic utility functions.
// Examples: string manipulation, time helpers, generic validation, slice operations.

// Example: ReverseString reverses a given string.
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// IsValidEmail checks if a string is a valid email address using a simple regex.
func IsValidEmail(email string) bool {
	// This is a basic regex for email validation. For production, consider a more robust solution.
	// For example, using a dedicated email validation library or a more comprehensive regex.
	match, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, email)
	return match
}

// ReadFileContent reads the content of a file at the given path and returns it as a string.
func ReadFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return string(content), nil
}
