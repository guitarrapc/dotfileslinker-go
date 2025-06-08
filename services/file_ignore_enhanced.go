package services

import (
	"strings"
)

// shouldIgnoreFileEnhanced determines whether a file should be ignored based on patterns.
// This is an enhanced version that properly handles negation patterns.
// filePath: The path to the file (relative to the repository root)
// fileName: The base name of the file
// isDir: Whether the path is a directory
// userIgnorePatterns: User-defined ignore patterns
func (s *FileLinkerService) shouldIgnoreFileEnhanced(filePath string, fileName string, isDir bool, userIgnorePatterns map[string]bool) bool {
	// Default state: don't ignore
	shouldIgnore := false

	// Check default ignore patterns (exact match)
	if _, exists := defaultIgnorePatterns[fileName]; exists {
		return true // Always ignore files that match default patterns
	}
	// Check for wildcards in default ignore patterns
	for pattern := range defaultIgnorePatterns {
		if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
			// For backward compatibility, we check fileName first
			if s.isAdvancedWildcardMatch(fileName, pattern) {
				return true // Always ignore files that match default patterns
			}
		}
	}

	// First pass: process non-negation patterns
	for pattern := range userIgnorePatterns {
		// Skip empty patterns and negation patterns for now
		if pattern == "" || strings.HasPrefix(pattern, "!") {
			continue
		}

		// Check exact match first
		if pattern == fileName {
			shouldIgnore = true
			continue
		}

		// Try with gitignore style matching for path patterns
		if strings.Contains(pattern, "/") || strings.Contains(pattern, "**") {
			if s.isGitIgnoreMatch(filePath, pattern, isDir) {
				shouldIgnore = true
				continue
			}
		}
		// For simple patterns or backward compatibility, try wildcards
		if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
			if s.isAdvancedWildcardMatch(fileName, pattern) {
				shouldIgnore = true
				continue
			}
		}
	}

	// Second pass: process negation patterns (these can override ignore decisions)
	for pattern := range userIgnorePatterns {
		// Only process negation patterns
		if !strings.HasPrefix(pattern, "!") {
			continue
		}

		// Remove the negation prefix for matching
		patternWithoutNegation := strings.TrimPrefix(pattern, "!")

		// Check if this negation pattern applies to our file
		matches := false

		// Try with gitignore style matching for path patterns
		if strings.Contains(patternWithoutNegation, "/") || strings.Contains(patternWithoutNegation, "**") {
			matches = s.isGitIgnoreMatch(filePath, patternWithoutNegation, isDir)
		} else if patternWithoutNegation == fileName {
			// Exact match
			matches = true
		} else if strings.Contains(patternWithoutNegation, "*") || strings.Contains(patternWithoutNegation, "?") {
			// Wildcard match
			matches = s.isAdvancedWildcardMatch(fileName, patternWithoutNegation)
		}

		// If the negation pattern matches, explicitly don't ignore this file
		if matches {
			shouldIgnore = false
		}
	}

	return shouldIgnore
}
