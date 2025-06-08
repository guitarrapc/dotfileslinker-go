package services

import (
	"strings"
)

// wildcard_matcher.go
// Implementation of improved wildcard pattern matching functions for file names

// isAdvancedWildcardMatch performs wildcard matching for file patterns
// Supporting multiple asterisks (*) and question marks (?) in patterns
func (s *FileLinkerService) isAdvancedWildcardMatch(text, pattern string) bool {
	// Case insensitive comparison
	text = strings.ToLower(text)
	pattern = strings.ToLower(pattern)

	// Edge cases
	if pattern == "" {
		return text == ""
	}
	if pattern == "*" {
		return true
	}

	// Use the gitIgnoreMatchPattern from gitignore_matcher.go to avoid code duplication
	return gitIgnoreMatchPattern(text, pattern, 0, 0)
}
