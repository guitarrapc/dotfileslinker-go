package service

import (
	"strings"
	"testing"
)

func TestShouldIgnoreFile(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		isDir    bool
		want     bool
	}{
		{name: "wildcard", patterns: []string{"*.log"}, path: "logs/app.log", want: true},
		{name: "question mark", patterns: []string{"file?.txt"}, path: "file1.txt", want: true},
		{name: "character class", patterns: []string{"file[0-9].txt"}, path: "file2.txt", want: true},
		{name: "double star", patterns: []string{"logs/**/*.log"}, path: "logs/2026/08/app.log", want: true},
		{name: "repository relative path", patterns: []string{"HOME/**/*.log"}, path: "HOME/logs/2026/app.log", want: true},
		{name: "directory contents", patterns: []string{"node_modules/"}, path: "node_modules/pkg/index.js", want: true},
		{name: "comment", patterns: []string{"# *.log"}, path: "app.log", want: false},
		{name: "escaped comment", patterns: []string{`\#notes.txt`}, path: "#notes.txt", want: true},
		{name: "case insensitive", patterns: []string{"file.txt"}, path: "FILE.txt", want: true},
		{name: "later negation wins", patterns: []string{"*.log", "!important.log"}, path: "important.log", want: false},
		{name: "later exclusion wins", patterns: []string{"!important.log", "*.log"}, path: "important.log", want: true},
		{name: "excluded parent cannot be rescued", patterns: []string{"logs/", "!logs/important.log"}, path: "logs/important.log", want: true},
		{name: "parent can be re-included", patterns: []string{"docs/", "!docs/", "docs/*", "!docs/README.md"}, path: "docs/README.md", want: false},
		{name: "other child remains ignored", patterns: []string{"docs/", "!docs/", "docs/*", "!docs/README.md"}, path: "docs/other.md", want: true},
		{name: "leading slash anchors to root", patterns: []string{"/config.json"}, path: "HOME/config.json", want: false},
		{name: "leading slash root match", patterns: []string{"/config.json"}, path: "config.json", want: true},
		{name: "trailing double star excludes descendants", patterns: []string{"logs/**"}, path: "logs/archive/app.log", want: true},
		{name: "trailing double star keeps directory", patterns: []string{"logs/**"}, path: "logs", isDir: true, want: false},
		{name: "unescaped trailing spaces", patterns: []string{"report.tmp   "}, path: "report.tmp", want: true},
		{name: "escaped trailing space", patterns: []string{`report.tmp\ `}, path: "report.tmp ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := newIgnoreMatcher(tt.patterns)
			if got := matcher.ignored(tt.path, tt.isDir); got != tt.want {
				t.Fatalf("ignored(%q, %v) = %v, want %v", tt.path, tt.isDir, got, tt.want)
			}
		})
	}
}

func TestDefaultIgnorePatterns(t *testing.T) {
	matcher := newIgnoreMatcher(nil)

	for _, path := range []string{".git/config", "nested/.svn/entries", "config.bak", "path/.file.swp"} {
		if !shouldIgnoreFile(path, false, matcher) {
			t.Errorf("default patterns should ignore %q", path)
		}
	}

	for _, path := range []string{".git", ".svn", ".hg"} {
		if shouldIgnoreFile(path, false, matcher) {
			t.Errorf("default VCS directory pattern should not ignore same-named file %q", path)
		}
		if !shouldIgnoreFile(path, true, matcher) {
			t.Errorf("default VCS directory pattern should ignore directory %q", path)
		}
	}
}

func TestDefaultIgnorePatternsCannotBeNegated(t *testing.T) {
	matcher := newIgnoreMatcher([]string{"!config.bak"})
	if !shouldIgnoreFile("config.bak", false, matcher) {
		t.Fatal("built-in ignore pattern was overridden by a user negation")
	}
}

func TestIgnoreMatcherDiscardsCommentsAndEmptyLines(t *testing.T) {
	matcher := newIgnoreMatcher([]string{"", "  ", "# comment", "*.tmp"})
	if got := matcher.count(); got != 1 {
		t.Fatalf("count() = %d, want 1", got)
	}
}

func TestWildcardMatch(t *testing.T) {
	tests := []struct {
		pattern string
		text    string
		want    bool
	}{
		{pattern: "a*c*g", text: "abcdefg", want: true},
		{pattern: "start*middle*end.txt", text: "start_wrong_end.txt", want: false},
		{pattern: "file?.txt", text: "file1.txt", want: true},
		{pattern: "file?.txt", text: "file12.txt", want: false},
		{pattern: "file[0-9].txt", text: "file7.txt", want: true},
		{pattern: "file[!0-9].txt", text: "filex.txt", want: true},
		{pattern: "file[!0-9].txt", text: "file7.txt", want: false},
		{pattern: `file\?.txt`, text: "file?.txt", want: true},
		{pattern: `file\*.txt`, text: "file*.txt", want: true},
		{pattern: "abc*ef", text: "AbCdEf", want: true},
	}

	for _, tt := range tests {
		if got := wildcardMatch(tt.text, tt.pattern); got != tt.want {
			t.Errorf("wildcardMatch(%q, %q) = %v, want %v", tt.text, tt.pattern, got, tt.want)
		}
	}
}

func TestMatchPathSegments(t *testing.T) {
	tests := []struct {
		name    string
		pattern []string
		value   string
		want    bool
	}{
		{name: "double star matches zero segments", pattern: []string{"logs", "**", "app.log"}, value: "logs/app.log", want: true},
		{name: "double star matches multiple segments", pattern: []string{"logs", "**", "app.log"}, value: "logs/2026/08/app.log", want: true},
		{name: "backtracks to double star", pattern: []string{"**", "cache", "target.txt"}, value: "cache/other/cache/target.txt", want: true},
		{name: "consecutive double stars", pattern: []string{"root", "**", "**", "target.txt"}, value: "root/a/b/target.txt", want: true},
		{name: "missing suffix", pattern: []string{"root", "**", "target.txt"}, value: "root/a/other.txt", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := pathSegmentsForTest(tt.value)
			if got := matchPathSegments(tt.pattern, tt.value, segments, len(segments)); got != tt.want {
				t.Errorf("matchPathSegments(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}

func TestMatchPathSegmentsAvoidsExponentialBacktracking(t *testing.T) {
	const count = 32
	pattern := make([]string, 0, count*2+1)
	value := strings.Repeat("segment/", count-1) + "segment"
	for range count {
		pattern = append(pattern, "**", "segment")
	}
	pattern = append(pattern, "missing")
	segments := pathSegmentsForTest(value)

	if matchPathSegments(pattern, value, segments, len(segments)) {
		t.Fatal("non-matching adversarial path unexpectedly matched")
	}

	if len(pattern) != count*2+1 {
		t.Fatal("adversarial pattern was constructed incorrectly")
	}
}

func TestIgnoreMatcherCommonPathDoesNotAllocate(t *testing.T) {
	matcher := newIgnoreMatcher([]string{"*.tmp", "HOME/**/cache/", "HOME/config/*.json"})
	allocations := testing.AllocsPerRun(1000, func() {
		if !matcher.ignored("HOME/config/settings.json", false) {
			t.Fatal("common path should match")
		}
	})
	if allocations != 0 {
		t.Fatalf("ignored() allocations = %v, want 0", allocations)
	}
}

func TestIgnoreMatcherHandlesPathDeeperThanStackSegmentLimit(t *testing.T) {
	matcher := newIgnoreMatcher([]string{"**/target.txt"})
	deepPath := strings.Repeat("level/", maxStackPathSegments+6) + "target.txt"
	if !matcher.ignored(deepPath, false) {
		t.Fatal("deep path should match")
	}
}

func pathSegmentsForTest(value string) []pathSegment {
	segments := make([]pathSegment, countPathSegments(value))
	fillPathSegments(value, segments)
	return segments
}
