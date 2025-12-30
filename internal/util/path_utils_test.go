package util

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
			name:     "Case difference (should be different on Unix, same on Windows)",
			pathA:    filepath.Join(os.TempDir(), "TEST"),
			pathB:    filepath.Join(os.TempDir(), "test"),
			expected: runtime.GOOS == "windows", // True on Windows, false on other platforms
		},
		{
			name:     "Paths with redundant elements",
			pathA:    filepath.Join(os.TempDir(), "test", ".."),
			pathB:    os.TempDir(),
			expected: true,
		},
		{
			name:     "Paths with trailing separators",
			pathA:    filepath.Join(os.TempDir(), "test") + string(os.PathSeparator),
			pathB:    filepath.Join(os.TempDir(), "test"),
			expected: true,
		},
		{
			name:     "Paths with different separators",
			pathA:    strings.ReplaceAll(filepath.Join(os.TempDir(), "test"), string(os.PathSeparator), "/"),
			pathB:    filepath.Join(os.TempDir(), "test"),
			expected: true,
		},
	}

	// Platform-specific test cases
	if runtime.GOOS == "windows" {
		windowsTests := []struct {
			name     string
			pathA    string
			pathB    string
			expected bool
		}{
			{
				name:     "Windows paths with different drive letter casing",
				pathA:    "C:\\temp\\test",
				pathB:    "c:\\temp\\test",
				expected: true, // Case-insensitive on Windows to match filesystem behavior
			},
			{
				name:     "Windows backslash vs forward slash",
				pathA:    "C:\\temp\\test\\path",
				pathB:    "C:/temp/test/path",
				expected: true, // Should normalize slashes
			},
		}
		tests = append(tests, windowsTests...)
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
