package services

import (
	"path/filepath"
	"strings"
)

// gitIgnorePattern represents a parsed .gitignore pattern
type gitIgnorePattern struct {
	raw      string   // Trimmed pattern string
	negation bool     // Whether the pattern starts with '!'
	dirOnly  bool     // Whether the pattern ends with '/' (directory only)
	segments []string // Path segments split by '/' (may include "**")
}

// isGitIgnoreMatch checks if a path matches a .gitignore style pattern
// - path: The path to check (using forward slashes)
// - pattern: The .gitignore style pattern
// - isDir: Whether the path represents a directory
func (s *FileLinkerService) isGitIgnoreMatch(path string, pattern string, isDir bool) bool {
	// Parse the pattern
	pat := parseGitIgnorePattern(pattern)

	// Skip directory-only patterns if the path is not a directory
	if pat.dirOnly && !isDir {
		return false
	}

	// Convert path to use forward slashes
	path = filepath.ToSlash(path)

	// Split path into segments
	pathSegs := strings.Split(path, "/")

	// Match the segments
	return matchSegments(pat.segments, pathSegs)
}

// parseGitIgnorePattern parses a .gitignore pattern string into a structured form
func parseGitIgnorePattern(pattern string) gitIgnorePattern {
	pat := gitIgnorePattern{raw: pattern}

	// Check for negation
	if strings.HasPrefix(pat.raw, "!") {
		pat.negation = true
		pat.raw = strings.TrimPrefix(pat.raw, "!")
		pat.raw = strings.TrimSpace(pat.raw)
	}

	// Check for directory-only pattern
	if strings.HasSuffix(pat.raw, "/") {
		pat.dirOnly = true
		pat.raw = strings.TrimSuffix(pat.raw, "/")
	}
	// Trim leading slash if present
	pat.raw = strings.TrimPrefix(pat.raw, "/")

	// Split into segments
	pat.segments = strings.Split(pat.raw, "/")

	return pat
}

// matchSegments checks if path segments match pattern segments
func matchSegments(segments, pathSegs []string) bool {
	return matchHelper(segments, pathSegs, 0, 0)
}

// matchHelper is a recursive helper for matchSegments
func matchHelper(segments, pathSegs []string, i, j int) bool {
	nSeg := len(segments)
	nPath := len(pathSegs)

	for i < nSeg && j < nPath {
		seg := segments[i]

		if seg == "**" {
			// "**" can match zero or more segments
			// If "**" is the last segment, match everything
			if i+1 == nSeg {
				return true
			}

			// Try to match the rest of the pattern at different positions
			for k := j; k <= nPath; k++ {
				if matchHelper(segments, pathSegs, i+1, k) {
					return true
				}
			}
			return false
		}

		// For non-"**" segments, match just one segment
		if !matchSingleSegment(seg, pathSegs[j]) {
			return false
		}

		i++
		j++
	}

	// After the loop: check if we've used all segments
	if i == nSeg && j == nPath {
		return true
	}

	// If we've consumed all path segments but still have pattern segments,
	// those remaining segments must all be "**"
	if i < nSeg && j == nPath {
		for k := i; k < nSeg; k++ {
			if segments[k] != "**" {
				return false
			}
		}
		return true
	}

	return false
}

// matchSingleSegment checks if a single path segment matches a pattern segment
func matchSingleSegment(segment, name string) bool {
	// Edge cases
	if segment == "" {
		return name == ""
	}
	if segment == "*" {
		return true
	}

	// For more complex patterns with * and ? wildcards
	return wildcardMatch(segment, name)
}

// wildcardMatch is a simple wildcard matcher for single segments
// Supports * (multiple chars) and ? (single char) wildcards
func wildcardMatch(pattern, text string) bool {
	// Case insensitive comparison
	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)

	return gitIgnoreMatchPattern(text, pattern, 0, 0)
}

// gitIgnoreMatchPattern is a helper function for recursive pattern matching
// This is a separate function to avoid name conflicts
func gitIgnoreMatchPattern(text, pattern string, ti, pi int) bool {
	textLen := len(text)
	patternLen := len(pattern)

	// Base case: if we've reached the end of both strings, we have a match
	if ti == textLen && pi == patternLen {
		return true
	}

	// If we've reached the end of the pattern but not the text, no match
	if pi == patternLen {
		return false
	}

	// Handle current pattern character
	switch pattern[pi] {
	case '*':
		// Try to match zero or more characters
		// 1) Skip the asterisk and try to match the rest of the pattern with the current text position
		// 2) Match one character and try again with the same pattern
		return gitIgnoreMatchPattern(text, pattern, ti, pi+1) ||
			(ti < textLen && gitIgnoreMatchPattern(text, pattern, ti+1, pi))
	case '?':
		// Match exactly one character
		return ti < textLen && gitIgnoreMatchPattern(text, pattern, ti+1, pi+1)
	default:
		// Match exact character
		return ti < textLen && pattern[pi] == text[ti] && gitIgnoreMatchPattern(text, pattern, ti+1, pi+1)
	}
}
