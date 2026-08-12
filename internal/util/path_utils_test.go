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

func TestLinkTargetEquals(t *testing.T) {
	root := filepath.Join(os.TempDir(), "dotfileslinker", "relative-link")
	linkPath := filepath.Join(root, "home", ".config", "settings.json")
	expectedTarget := filepath.Join(root, "repo", "settings.json")
	relativeTarget, err := filepath.Rel(filepath.Dir(linkPath), expectedTarget)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		linkPath       string
		linkTarget     string
		expectedTarget string
		want           bool
	}{
		{name: "relative target", linkPath: linkPath, linkTarget: relativeTarget, expectedTarget: expectedTarget, want: true},
		{name: "absolute target", linkPath: linkPath, linkTarget: expectedTarget, expectedTarget: expectedTarget, want: true},
		{name: "different target", linkPath: linkPath, linkTarget: relativeTarget, expectedTarget: filepath.Join(root, "repo", "other.json"), want: false},
		{name: "empty link path", linkTarget: relativeTarget, expectedTarget: expectedTarget, want: false},
		{name: "empty link target", linkPath: linkPath, expectedTarget: expectedTarget, want: false},
		{name: "empty expected target", linkPath: linkPath, linkTarget: relativeTarget, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LinkTargetEquals(tt.linkPath, tt.linkTarget, tt.expectedTarget); got != tt.want {
				t.Errorf("LinkTargetEquals(%q, %q, %q) = %v, want %v", tt.linkPath, tt.linkTarget, tt.expectedTarget, got, tt.want)
			}
		})
	}
}

func TestPathContainment(t *testing.T) {
	root := filepath.Join(os.TempDir(), "dotfileslinker", "containment")
	tests := []struct {
		name        string
		path        string
		directory   string
		wantInside  bool
		wantOverlap bool
	}{
		{name: "same", path: root, directory: root, wantInside: true, wantOverlap: true},
		{name: "descendant", path: filepath.Join(root, "nested", "file"), directory: root, wantInside: true, wantOverlap: true},
		{name: "ancestor", path: root, directory: filepath.Join(root, "nested"), wantInside: false, wantOverlap: true},
		{name: "sibling prefix", path: root + "-other", directory: root, wantInside: false, wantOverlap: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSameOrDescendant(tt.path, tt.directory); got != tt.wantInside {
				t.Errorf("IsSameOrDescendant(%q, %q) = %v, want %v", tt.path, tt.directory, got, tt.wantInside)
			}
			if got := PathsOverlap(tt.path, tt.directory); got != tt.wantOverlap {
				t.Errorf("PathsOverlap(%q, %q) = %v, want %v", tt.path, tt.directory, got, tt.wantOverlap)
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
