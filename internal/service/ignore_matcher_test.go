package service

import (
	"testing"

	gitignore "github.com/idelchi/go-gitignore"
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
		{name: "case sensitive", patterns: []string{"file.txt"}, path: "FILE.txt", want: false},
		{name: "later negation wins", patterns: []string{"*.log", "!important.log"}, path: "important.log", want: false},
		{name: "later exclusion wins", patterns: []string{"!important.log", "*.log"}, path: "important.log", want: true},
		{name: "excluded parent cannot be rescued", patterns: []string{"logs/", "!logs/important.log"}, path: "logs/important.log", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := gitignore.New(tt.patterns...)
			if got := shouldIgnoreFile(tt.path, tt.isDir, matcher); got != tt.want {
				t.Fatalf("shouldIgnoreFile(%q, %v) = %v, want %v", tt.path, tt.isDir, got, tt.want)
			}
		})
	}
}

func TestDefaultIgnorePatterns(t *testing.T) {
	matcher := gitignore.New(defaultIgnorePatterns...)

	for _, path := range []string{".git/config", "nested/.svn/entries", "config.bak", "path/.file.swp"} {
		if !shouldIgnoreFile(path, false, matcher) {
			t.Errorf("default patterns should ignore %q", path)
		}
	}
}
