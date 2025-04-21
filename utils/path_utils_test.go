package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathEquals(t *testing.T) {
	// Define test cases
	tests := []struct {
		name     string
		pathA    string
		pathB    string
		expected bool
	}{
		{
			name:     "Same absolute paths",
			pathA:    filepath.Join(os.TempDir(), "test"),
			pathB:    filepath.Join(os.TempDir(), "test"),
			expected: true,
		},
		{
			name:     "Different absolute paths",
			pathA:    filepath.Join(os.TempDir(), "test1"),
			pathB:    filepath.Join(os.TempDir(), "test2"),
			expected: false,
		},
		{
			name:     "Relative and absolute paths",
			pathA:    "./test",
			pathB:    filepath.Join(mustGetwd(), "test"),
			expected: true,
		},
		{
			name:     "Case difference (should be different)",
			pathA:    filepath.Join(os.TempDir(), "TEST"),
			pathB:    filepath.Join(os.TempDir(), "test"),
			expected: false,
		},
	}

	// Execute test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PathEquals(tt.pathA, tt.pathB)
			if result != tt.expected {
				t.Errorf("PathEquals(%q, %q) = %v; want %v", tt.pathA, tt.pathB, result, tt.expected)
			}
		})
	}
}

// Test helper: Get current working directory and panic on error
func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return dir
}
