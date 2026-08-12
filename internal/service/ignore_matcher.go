package service

import (
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ignoreMatcher implements the subset of .gitignore syntax documented by this
// application. Patterns are compiled once and evaluated in source order.
type ignoreMatcher struct {
	patterns []ignorePattern
}

type ignorePattern struct {
	segments           []string
	negated            bool
	directoryOnly      bool
	anchored           bool
	hasSlash           bool
	trailingDoubleStar bool
}

func newIgnoreMatcher(lines []string) *ignoreMatcher {
	m := &ignoreMatcher{patterns: make([]ignorePattern, 0, len(lines))}
	m.append(lines)
	return m
}

func (m *ignoreMatcher) count() int {
	return len(m.patterns)
}

func (m *ignoreMatcher) append(lines []string) {
	for _, line := range lines {
		if pattern, ok := parseIgnorePattern(line); ok {
			m.patterns = append(m.patterns, pattern)
		}
	}
}

func parseIgnorePattern(line string) (ignorePattern, bool) {
	if strings.TrimSpace(line) == "" {
		return ignorePattern{}, false
	}

	line = trimUnescapedTrailingSpaces(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ignorePattern{}, false
	}

	pattern := ignorePattern{}
	switch {
	case strings.HasPrefix(line, `\#`), strings.HasPrefix(line, `\!`):
		line = line[1:]
	case strings.HasPrefix(line, "!"):
		pattern.negated = true
		line = line[1:]
	}

	if line == "" {
		return ignorePattern{}, false
	}

	pattern.anchored = strings.HasPrefix(line, "/")
	pattern.directoryOnly = strings.HasSuffix(line, "/")
	line = strings.Trim(line, "/")
	pattern.hasSlash = strings.Contains(line, "/")
	if line == "" {
		return ignorePattern{}, false
	}

	pattern.segments = strings.Split(line, "/")
	pattern.trailingDoubleStar = len(pattern.segments) > 1 && pattern.segments[len(pattern.segments)-1] == "**"
	return pattern, true
}

func trimUnescapedTrailingSpaces(value string) string {
	for len(value) > 0 && value[len(value)-1] == ' ' {
		backslashes := 0
		for i := len(value) - 2; i >= 0 && value[i] == '\\'; i-- {
			backslashes++
		}
		if backslashes%2 == 1 {
			return value[:len(value)-2] + value[len(value)-1:]
		}
		value = value[:len(value)-1]
	}
	return value
}

func (m *ignoreMatcher) ignored(filePath string, isDir bool) bool {
	filePath = strings.TrimLeft(path.Clean(filePath), "/")
	if filePath == "." || filePath == "" {
		return false
	}

	pathSegments := strings.Split(filePath, "/")
	for end := 1; end < len(pathSegments); end++ {
		if m.directlyIgnored(pathSegments[:end], true) {
			return true
		}
	}
	return m.directlyIgnored(pathSegments, isDir)
}

func (m *ignoreMatcher) directlyIgnored(pathSegments []string, isDir bool) bool {
	ignored := false
	for _, pattern := range m.patterns {
		if pattern.matches(pathSegments, isDir) {
			ignored = !pattern.negated
		}
	}
	return ignored
}

func (p ignorePattern) matches(pathSegments []string, isDir bool) bool {
	if p.directoryOnly && !isDir {
		return false
	}

	if !p.hasSlash && !p.anchored {
		return wildcardMatch(pathSegments[len(pathSegments)-1], p.segments[0])
	}

	if !p.hasSlash && p.anchored && len(pathSegments) != 1 {
		return false
	}

	if p.trailingDoubleStar && len(pathSegments) < len(p.segments) {
		return false
	}

	return matchPathSegments(p.segments, pathSegments)
}

func matchPathSegments(pattern, value []string) bool {
	patternIndex, valueIndex := 0, 0
	// Remember the most recent ** checkpoint. On a later mismatch, let that
	// ** consume one more segment instead of recursively trying every split.
	doubleStarPatternIndex, doubleStarValueIndex := -1, -1

	for valueIndex < len(value) {
		if patternIndex < len(pattern) && pattern[patternIndex] == "**" {
			for patternIndex < len(pattern) && pattern[patternIndex] == "**" {
				patternIndex++
			}
			doubleStarPatternIndex = patternIndex
			doubleStarValueIndex = valueIndex
			continue
		}

		if patternIndex < len(pattern) && wildcardMatch(value[valueIndex], pattern[patternIndex]) {
			patternIndex++
			valueIndex++
			continue
		}

		if doubleStarPatternIndex < 0 || doubleStarValueIndex >= len(value) {
			return false
		}
		doubleStarValueIndex++
		valueIndex = doubleStarValueIndex
		patternIndex = doubleStarPatternIndex
	}

	for patternIndex < len(pattern) && pattern[patternIndex] == "**" {
		patternIndex++
	}
	return patternIndex == len(pattern)
}

// wildcardMatch matches one path segment without allocating temporary strings.
func wildcardMatch(text, pattern string) bool {
	textIndex, patternIndex := 0, 0
	starPatternIndex, starTextIndex := -1, -1

	for textIndex < len(text) {
		if patternIndex < len(pattern) && pattern[patternIndex] == '*' {
			for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
				patternIndex++
			}
			starPatternIndex = patternIndex
			starTextIndex = textIndex
			continue
		}

		value, valueSize := utf8.DecodeRuneInString(text[textIndex:])
		if nextPatternIndex, ok := matchWildcardToken(pattern, patternIndex, value); ok {
			patternIndex = nextPatternIndex
			textIndex += valueSize
			continue
		}

		if starPatternIndex < 0 || starTextIndex >= len(text) {
			return false
		}
		_, size := utf8.DecodeRuneInString(text[starTextIndex:])
		starTextIndex += size
		textIndex = starTextIndex
		patternIndex = starPatternIndex
	}

	for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
		patternIndex++
	}
	return patternIndex == len(pattern)
}

func matchWildcardToken(pattern string, index int, value rune) (int, bool) {
	if index >= len(pattern) {
		return index, false
	}

	token, tokenSize := utf8.DecodeRuneInString(pattern[index:])
	switch token {
	case '?':
		return index + tokenSize, true
	case '\\':
		nextIndex := index + tokenSize
		if nextIndex >= len(pattern) {
			return index + tokenSize, equalFoldRune(token, value)
		}
		escaped, escapedSize := utf8.DecodeRuneInString(pattern[nextIndex:])
		return nextIndex + escapedSize, equalFoldRune(escaped, value)
	case '[':
		if nextIndex, matched, ok := matchCharacterClass(pattern, index, value); ok {
			return nextIndex, matched
		}
	}

	return index + tokenSize, equalFoldRune(token, value)
}

func matchCharacterClass(pattern string, start int, value rune) (next int, matched bool, ok bool) {
	index := start + 1
	negated := index < len(pattern) && (pattern[index] == '!' || pattern[index] == '^')
	if negated {
		index++
	}
	classStart := index

	for index < len(pattern) && pattern[index] != ']' {
		lower, lowerSize := utf8.DecodeRuneInString(pattern[index:])
		index += lowerSize
		if lower == '\\' && index < len(pattern) {
			lower, lowerSize = utf8.DecodeRuneInString(pattern[index:])
			index += lowerSize
		}

		if index < len(pattern) && pattern[index] == '-' && index+1 < len(pattern) && pattern[index+1] != ']' {
			index++
			upper, upperSize := utf8.DecodeRuneInString(pattern[index:])
			index += upperSize
			matched = matched || inFoldedRange(value, lower, upper)
		} else {
			matched = matched || equalFoldRune(lower, value)
		}
	}

	if index >= len(pattern) || index == classStart {
		return start + 1, false, false
	}
	if negated {
		matched = !matched
	}
	return index + 1, matched, true
}

func equalFoldRune(left, right rune) bool {
	return unicode.ToUpper(left) == unicode.ToUpper(right)
}

func inFoldedRange(value, lower, upper rune) bool {
	value = unicode.ToUpper(value)
	lower = unicode.ToUpper(lower)
	upper = unicode.ToUpper(upper)
	return value >= lower && value <= upper
}
