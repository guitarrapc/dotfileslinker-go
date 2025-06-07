package utils

import (
	"path/filepath"
	"runtime"
	"strings"
)

// PathEquals compares two file or directory paths for equality by resolving to absolute paths.
// This function performs a platform-specific comparison:
// - On Windows: case-insensitive comparison (matching the filesystem behavior)
// - On other platforms (Linux, macOS): case-sensitive comparison
//
// The function normalizes directory separators and removes redundant path elements
// before comparison to ensure consistent results.
func PathEquals(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)

	if errA != nil || errB != nil {
		return false
	}

	// Clean paths to normalize directory separators and remove redundant elements
	cleanA := filepath.Clean(absA)
	cleanB := filepath.Clean(absB)

	// On Windows, perform case-insensitive comparison
	if runtime.GOOS == "windows" {
		return strings.EqualFold(cleanA, cleanB)
	}

	// On other platforms, perform case-sensitive comparison
	return cleanA == cleanB
}
