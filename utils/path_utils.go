package utils

import (
	"path/filepath"
)

// PathEquals compares two file or directory paths for equality by resolving to absolute paths.
// This function performs a case-sensitive comparison on all platforms, including Windows,
// which may result in re-linking files that differ only in case on case-insensitive filesystems.
//
// Note: Although Windows filesystem is case-insensitive, we intentionally use case-sensitive
// comparison for consistency across platforms and to encourage proper casing in path references.
// Since re-linking has minimal cost, this approach ensures paths are exactly matched and
// potential casing discrepancies are corrected during the linking process.
func PathEquals(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)

	if errA != nil || errB != nil {
		return false
	}

	// Perform case-sensitive comparison on all platforms
	return absA == absB
}
