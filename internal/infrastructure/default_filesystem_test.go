package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnumerateDirectoriesReturnsImmediateChildren(t *testing.T) {
	root := t.TempDir()
	keptDirectory := filepath.Join(root, "kept")
	skippedDirectory := filepath.Join(root, "ignored")
	if err := os.MkdirAll(keptDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(skippedDirectory, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	fs := NewDefaultFileSystem()
	directories, err := fs.EnumerateDirectories(root)
	if err != nil {
		t.Fatalf("EnumerateDirectories() error = %v", err)
	}
	if len(directories) != 2 {
		t.Fatalf("EnumerateDirectories() = %v, want two immediate children", directories)
	}
	for _, nested := range directories {
		if nested == filepath.Join(skippedDirectory, "nested") {
			t.Fatalf("nested directory was returned: %v", directories)
		}
	}
}
