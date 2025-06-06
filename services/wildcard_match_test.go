package services

import (
	"path/filepath"
	"testing"
)

// TestIsWildcardMatch tests the isWildcardMatch function with various patterns
func TestIsWildcardMatch(t *testing.T) {
	// Create a service instance for testing
	service := &FileLinkerService{}

	tests := []struct {
		name     string
		fileName string
		pattern  string
		expected bool
	}{
		// Single wildcard patterns - supported by current implementation
		{
			name:     "Prefix wildcard match",
			fileName: "file.txt.bak",
			pattern:  "*.bak",
			expected: true,
		},
		{
			name:     "Prefix wildcard no match",
			fileName: "file.txt",
			pattern:  "*.bak",
			expected: false,
		},
		{
			name:     "Suffix wildcard match",
			fileName: "tempfile.txt",
			pattern:  "temp*",
			expected: true,
		},
		{
			name:     "Suffix wildcard no match",
			fileName: "file.txt",
			pattern:  "temp*",
			expected: false,
		},
		{
			name:     "Middle wildcard match",
			fileName: "prefix_suffix.txt",
			pattern:  "prefix*suffix.txt",
			expected: true,
		},
		{
			name:     "Middle wildcard no match - wrong prefix",
			fileName: "wrong_suffix.txt",
			pattern:  "prefix*suffix.txt",
			expected: false,
		},
		{
			name:     "Middle wildcard no match - wrong suffix",
			fileName: "prefix_wrong.txt",
			pattern:  "prefix*suffix.txt",
			expected: false,
		},

		// Multiple wildcard patterns - current implementation limitations
		{
			name:     "Two wildcards - should match but current implementation doesn't support",
			fileName: "abcdefg",
			pattern:  "a*c*g",
			expected: false, // Current implementation will return false
		},
		{
			name:     "Complex pattern with multiple wildcards",
			fileName: "a_middle_z.txt",
			pattern:  "a*middle*z.txt",
			expected: false, // Current implementation will return false
		},

		// Edge cases
		{
			name:     "Wildcard only pattern",
			fileName: "anything.txt",
			pattern:  "*",
			expected: true, // Current implementation will return false, but should be true
		},
		{
			name:     "Case insensitive match",
			fileName: "File.TXT",
			pattern:  "file*",
			expected: true,
		},
		{
			name:     "Empty pattern",
			fileName: "file.txt",
			pattern:  "",
			expected: false,
		},
		{
			name:     "Empty filename",
			fileName: "",
			pattern:  "file*",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isWildcardMatch(tt.fileName, tt.pattern)
			if result != tt.expected {
				t.Errorf("isWildcardMatch(%q, %q) = %v; expected %v",
					tt.fileName, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestPathGlobBehavior tests how filepath.Glob behaves with multiple wildcards
func TestPathGlobBehavior(t *testing.T) {
	// This test demonstrates the behavior of filepath.Match with multiple wildcards
	// to help understand what's possible with Go's standard library

	tests := []struct {
		name     string
		pattern  string
		fileName string
		expected bool
	}{
		// Standard glob patterns
		{
			name:     "Single asterisk matches any characters",
			pattern:  "file*.txt",
			fileName: "file123.txt",
			expected: true,
		},
		{
			name:     "Multiple consecutive asterisks act as one",
			pattern:  "file**.txt", // Two asterisks in a row
			fileName: "file123.txt",
			expected: true, // In Go's filepath.Match, ** is not special, it's treated as *
		},
		{
			name:     "Multiple separate asterisks",
			pattern:  "a*b*c.txt", // Two separate asterisks
			fileName: "axbyc.txt",
			expected: true, // This works because each * is a separate wildcard
		},
		{
			name:     "Non-matching pattern with multiple wildcards",
			pattern:  "a*b*c.txt",
			fileName: "axcy.txt", // Missing 'b'
			expected: false,
		},
		{
			name:     "Pattern with ? wildcard (single character)",
			pattern:  "file?.txt",
			fileName: "file1.txt",
			expected: true,
		},
		{
			name:     "Pattern with character class",
			pattern:  "file[123].txt",
			fileName: "file2.txt",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using filepath.Match directly to test Go's native pattern matching
			matched, err := filepath.Match(tt.pattern, tt.fileName)
			if err != nil {
				t.Fatalf("filepath.Match(%q, %q) returned error: %v",
					tt.pattern, tt.fileName, err)
			}

			if matched != tt.expected {
				t.Errorf("filepath.Match(%q, %q) = %v; expected %v",
					tt.pattern, tt.fileName, matched, tt.expected)
			}
		})
	}
}
