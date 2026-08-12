package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnumerateFilesSkipsDirectoryTraversal(t *testing.T) {
	root := t.TempDir()
	keptDirectory := filepath.Join(root, "kept")
	skippedDirectory := filepath.Join(root, "ignored")
	if err := os.MkdirAll(keptDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(skippedDirectory, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	keptFile := filepath.Join(keptDirectory, "keep.txt")
	if err := os.WriteFile(keptFile, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skippedDirectory, "nested", "skip.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := NewDefaultFileSystem()
	visitedSkippedDirectory := false
	files, err := fs.EnumerateFiles(root, "*", true, func(directory string) bool {
		if directory == skippedDirectory {
			visitedSkippedDirectory = true
			return true
		}
		return false
	})
	if err != nil {
		t.Fatalf("EnumerateFiles() error = %v", err)
	}
	if !visitedSkippedDirectory {
		t.Fatal("skip callback was not called for ignored directory")
	}
	if len(files) != 1 || files[0] != keptFile {
		t.Fatalf("EnumerateFiles() = %v, want [%s]", files, keptFile)
	}
}
