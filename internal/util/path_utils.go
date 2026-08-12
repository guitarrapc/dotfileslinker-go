package util

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

	return pathsEqual(cleanA, cleanB)
}

// LinkTargetEquals compares a symbolic link target with an expected path.
// Relative link targets are resolved from the symbolic link's parent directory.
func LinkTargetEquals(linkPath, linkTarget, expectedTarget string) bool {
	if linkPath == "" || linkTarget == "" || expectedTarget == "" {
		return false
	}

	fullLinkPath, err := filepath.Abs(linkPath)
	if err != nil {
		return false
	}
	resolvedLinkTarget := linkTarget
	if !filepath.IsAbs(resolvedLinkTarget) {
		resolvedLinkTarget = filepath.Join(filepath.Dir(fullLinkPath), resolvedLinkTarget)
	}

	fullLinkTarget, err := filepath.Abs(resolvedLinkTarget)
	if err != nil {
		return false
	}
	fullExpectedTarget, err := filepath.Abs(expectedTarget)
	if err != nil {
		return false
	}

	return pathsEqual(filepath.Clean(fullLinkTarget), filepath.Clean(fullExpectedTarget))
}

func pathsEqual(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
