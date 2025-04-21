package utils

import (
	"path/filepath"
)

// PathEquals compares two file or directory paths for equality, considering their absolute paths.
// This function resolves the absolute paths of the input paths and performs a case-sensitive comparison.
func PathEquals(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)

	if errA != nil || errB != nil {
		return false
	}

	return absA == absB
}
