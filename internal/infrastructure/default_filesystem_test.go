package infrastructure

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestFileAndDirectoryExistsReflectActualFileSystem(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	directory := filepath.Join(root, "directory")
	fs := NewDefaultFileSystem()

	if fs.FileExists(file) || fs.DirectoryExists(directory) {
		t.Fatal("missing paths were reported as existing")
	}
	if err := os.WriteFile(file, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}

	if !fs.FileExists(file) {
		t.Fatal("FileExists() = false for existing file")
	}
	if fs.FileExists(directory) {
		t.Fatal("FileExists() = true for directory")
	}
	if !fs.DirectoryExists(directory) {
		t.Fatal("DirectoryExists() = false for existing directory")
	}
	if fs.DirectoryExists(file) {
		t.Fatal("DirectoryExists() = true for file")
	}
}

func TestEnsureDirectoryCreatesParentsAndIsIdempotent(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "parent", "child")
	fs := NewDefaultFileSystem()

	if err := fs.EnsureDirectory(directory); err != nil {
		t.Fatalf("first EnsureDirectory() error = %v", err)
	}
	if err := fs.EnsureDirectory(directory); err != nil {
		t.Fatalf("second EnsureDirectory() error = %v", err)
	}
	if !fs.DirectoryExists(directory) {
		t.Fatal("EnsureDirectory() did not create directory")
	}
}

func TestReadAllLinesReadsLFAndCRLF(t *testing.T) {
	root := t.TempDir()
	fs := NewDefaultFileSystem()
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "LF", content: "one\ntwo", want: []string{"one", "two"}},
		{name: "CRLF", content: "one\r\ntwo", want: []string{"one", "two"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(root, tt.name+".txt")
			if err := os.WriteFile(file, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := fs.ReadAllLines(file)
			if err != nil {
				t.Fatalf("ReadAllLines() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ReadAllLines() = %q, want %q", got, tt.want)
			}
		})
	}

	if _, err := fs.ReadAllLines(filepath.Join(root, "missing.txt")); err == nil {
		t.Fatal("ReadAllLines() returned nil error for missing file")
	}
}

func TestEnumerateFilesHonorsPatternAndRecursion(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	rootText := writeTestFile(t, filepath.Join(root, "root.txt"), "root")
	writeTestFile(t, filepath.Join(root, "root.json"), "json")
	nestedText := writeTestFile(t, filepath.Join(nested, "nested.txt"), "nested")
	fs := NewDefaultFileSystem()

	topLevel, err := fs.EnumerateFiles(root, "*.txt", false)
	if err != nil {
		t.Fatalf("non-recursive EnumerateFiles() error = %v", err)
	}
	recursive, err := fs.EnumerateFiles(root, "*.txt", true)
	if err != nil {
		t.Fatalf("recursive EnumerateFiles() error = %v", err)
	}
	assertPathsEqual(t, topLevel, []string{rootText})
	assertPathsEqual(t, recursive, []string{rootText, nestedText})

	if _, err := fs.EnumerateFiles(root, "[", true); err == nil {
		t.Fatal("EnumerateFiles() returned nil error for invalid pattern")
	}
}

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
	writeTestFile(t, filepath.Join(root, "file.txt"), "content")
	fs := NewDefaultFileSystem()
	directories, err := fs.EnumerateDirectories(root)
	if err != nil {
		t.Fatalf("EnumerateDirectories() error = %v", err)
	}
	assertPathsEqual(t, directories, []string{keptDirectory, skippedDirectory})
}

func TestReadDirectoryReturnsImmediateChildren(t *testing.T) {
	root := t.TempDir()
	file := writeTestFile(t, filepath.Join(root, "file.txt"), "content")
	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(directory, "nested.txt"), "nested")

	entries, err := NewDefaultFileSystem().ReadDirectory(root)
	if err != nil {
		t.Fatalf("ReadDirectory() error = %v", err)
	}
	want := map[string]bool{file: false, directory: true}
	if len(entries) != len(want) {
		t.Fatalf("ReadDirectory() returned %d entries, want %d: %v", len(entries), len(want), entries)
	}
	for _, entry := range entries {
		isDirectory, exists := want[entry.Path]
		if !exists {
			t.Errorf("unexpected entry: %+v", entry)
			continue
		}
		if entry.IsDirectory != isDirectory {
			t.Errorf("entry %s IsDirectory = %v, want %v", entry.Path, entry.IsDirectory, isDirectory)
		}
	}
}

func TestCreateFileSymlinkAndGetLinkTarget(t *testing.T) {
	root := t.TempDir()
	target := writeTestFile(t, filepath.Join(root, "targets", "file.txt"), "content")
	link := filepath.Join(root, "file-link.txt")
	fs := NewDefaultFileSystem()

	if err := fs.CreateFileSymlink(link, target); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if got := fs.GetLinkTarget(link); got != target {
		t.Fatalf("GetLinkTarget() = %q, want %q", got, target)
	}
	content, err := os.ReadFile(link)
	if err != nil {
		t.Fatalf("reading through link: %v", err)
	}
	if string(content) != "content" {
		t.Fatalf("content through link = %q", content)
	}
}

func TestGetLinkTargetPreservesRelativeTarget(t *testing.T) {
	root := t.TempDir()
	target := writeTestFile(t, filepath.Join(root, "targets", "relative.txt"), "content")
	linksDirectory := filepath.Join(root, "links")
	if err := os.Mkdir(linksDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(linksDirectory, "relative-link.txt")
	relativeTarget, err := filepath.Rel(linksDirectory, target)
	if err != nil {
		t.Fatal(err)
	}
	fs := NewDefaultFileSystem()

	if err := fs.CreateFileSymlink(link, relativeTarget); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if got := fs.GetLinkTarget(link); got != relativeTarget {
		t.Fatalf("GetLinkTarget() = %q, want stored relative target %q", got, relativeTarget)
	}
	if content, err := os.ReadFile(link); err != nil || string(content) != "content" {
		t.Fatalf("relative link did not resolve: content = %q, error = %v", content, err)
	}
}

func TestCreateDirectorySymlinkAndGetLinkTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target-directory")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(target, "file.txt"), "content")
	link := filepath.Join(root, "directory-link")
	fs := NewDefaultFileSystem()

	if err := fs.CreateDirectorySymlink(link, target); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if got := fs.GetLinkTarget(link); got != target {
		t.Fatalf("GetLinkTarget() = %q, want %q", got, target)
	}
	if content, err := os.ReadFile(filepath.Join(link, "file.txt")); err != nil || string(content) != "content" {
		t.Fatalf("directory link did not resolve: content = %q, error = %v", content, err)
	}
}

func TestDeleteRemovesEntriesAndPreservesSymlinkTarget(t *testing.T) {
	root := t.TempDir()
	fs := NewDefaultFileSystem()
	file := writeTestFile(t, filepath.Join(root, "delete.txt"), "delete")
	directory := filepath.Join(root, "empty")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.Delete(file); err != nil {
		t.Fatalf("Delete(file) error = %v", err)
	}
	if err := fs.Delete(directory); err != nil {
		t.Fatalf("Delete(directory) error = %v", err)
	}
	if _, err := os.Lstat(file); !os.IsNotExist(err) {
		t.Fatalf("deleted file still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Lstat(directory); !os.IsNotExist(err) {
		t.Fatalf("deleted directory still exists or stat failed unexpectedly: %v", err)
	}

	target := writeTestFile(t, filepath.Join(root, "preserved.txt"), "preserved")
	link := filepath.Join(root, "preserved-link.txt")
	if err := fs.CreateFileSymlink(link, target); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if err := fs.Delete(link); err != nil {
		t.Fatalf("Delete(link) error = %v", err)
	}
	if !fs.FileExists(target) {
		t.Fatal("Delete(link) removed link target")
	}
	exists, err := fs.PathExists(link)
	if err != nil || exists {
		t.Fatalf("link still exists after Delete(): exists = %v, error = %v", exists, err)
	}
}

func TestRemoveAllRemovesNonEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	fs := NewDefaultFileSystem()
	directory := filepath.Join(root, "non-empty")
	writeTestFile(t, filepath.Join(directory, "nested", "file.txt"), "content")

	if err := fs.RemoveAll(directory); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}
	if _, err := os.Lstat(directory); !os.IsNotExist(err) {
		t.Fatalf("removed directory still exists or stat failed unexpectedly: %v", err)
	}
}

func TestDeleteBackupRemovesGeneratedNonEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "target")
	backup := original + ".dotfileslinker-backup.1"
	writeTestFile(t, filepath.Join(backup, "nested", "file.txt"), "content")

	if err := NewDefaultFileSystem().DeleteBackup(backup, original); err != nil {
		t.Fatalf("DeleteBackup() error = %v", err)
	}
	if _, err := os.Lstat(backup); !os.IsNotExist(err) {
		t.Fatalf("backup still exists or stat failed unexpectedly: %v", err)
	}
}

func TestDeleteBackupRejectsUnrelatedPath(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "target")
	unrelated := filepath.Join(root, "unrelated")
	file := writeTestFile(t, filepath.Join(unrelated, "file.txt"), "preserved")

	if err := NewDefaultFileSystem().DeleteBackup(unrelated, original); err == nil {
		t.Fatal("DeleteBackup() accepted an unrelated path")
	}
	if content, err := os.ReadFile(file); err != nil || string(content) != "preserved" {
		t.Fatalf("unrelated path was modified: content = %q, error = %v", content, err)
	}
}

func TestDeleteBackupRejectsInvalidNumericSuffix(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "target")
	for _, suffix := range []string{".0", ".-1", ".1-extra", ".not-a-number"} {
		backup := original + ".dotfileslinker-backup" + suffix
		if err := NewDefaultFileSystem().DeleteBackup(backup, original); err == nil {
			t.Errorf("DeleteBackup() accepted invalid suffix %q", suffix)
		}
	}
}

func TestMoveRenamesSymlinkWithoutMovingTarget(t *testing.T) {
	root := t.TempDir()
	fs := NewDefaultFileSystem()
	target := writeTestFile(t, filepath.Join(root, "target.txt"), "preserved")
	link := filepath.Join(root, "link.txt")
	movedLink := filepath.Join(root, "moved-link.txt")
	if err := fs.CreateFileSymlink(link, target); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}

	if err := fs.Move(link, movedLink); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if got := fs.GetLinkTarget(movedLink); got != target {
		t.Fatalf("moved link target = %q, want %q", got, target)
	}
	if !fs.FileExists(target) {
		t.Fatal("Move(link) moved or removed the link target")
	}
	exists, err := fs.PathExists(link)
	if err != nil || exists {
		t.Fatalf("original link remains after Move(): exists = %v, error = %v", exists, err)
	}
}

func TestGetLinkTargetReturnsEmptyForNonLinks(t *testing.T) {
	root := t.TempDir()
	file := writeTestFile(t, filepath.Join(root, "regular.txt"), "content")
	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	fs := NewDefaultFileSystem()
	if got := fs.GetLinkTarget(file); got != "" {
		t.Fatalf("GetLinkTarget(file) = %q, want empty", got)
	}
	if got := fs.GetLinkTarget(directory); got != "" {
		t.Fatalf("GetLinkTarget(directory) = %q, want empty", got)
	}
}

func TestPathExistsFindsDanglingSymlink(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "dangling")
	if err := os.Symlink(filepath.Join(root, "missing"), link); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}

	exists, err := NewDefaultFileSystem().PathExists(link)
	if err != nil {
		t.Fatalf("PathExists() error = %v", err)
	}
	if !exists {
		t.Fatal("PathExists() = false for dangling symbolic link")
	}
}

func TestPathExistsReturnsFalseForMissingPath(t *testing.T) {
	exists, err := NewDefaultFileSystem().PathExists(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("PathExists() error = %v", err)
	}
	if exists {
		t.Fatal("PathExists() = true for missing path")
	}
}

func writeTestFile(t *testing.T, file, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return file
}

func assertPathsEqual(t *testing.T, got, want []string) {
	t.Helper()
	got = append([]string(nil), got...)
	want = append([]string(nil), want...)
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
}
