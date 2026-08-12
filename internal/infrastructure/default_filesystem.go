package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// DefaultFileSystem provides the default implementation of the FileSystem interface.
type DefaultFileSystem struct{}

// NewDefaultFileSystem creates a new instance of DefaultFileSystem.
func NewDefaultFileSystem() *DefaultFileSystem {
	return &DefaultFileSystem{}
}

// FileExists determines whether the specified file exists.
func (dfs *DefaultFileSystem) FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirectoryExists determines whether the specified directory exists.
func (dfs *DefaultFileSystem) DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// PathExists determines whether a directory entry exists without following symbolic links.
func (dfs *DefaultFileSystem) PathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetLinkTarget gets the target of a symbolic link at the specified path.
func (dfs *DefaultFileSystem) GetLinkTarget(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return ""
	}
	return target
}

// Delete deletes the specified file or empty directory.
func (dfs *DefaultFileSystem) Delete(path string) error {
	return os.Remove(path)
}

// RemoveAll deletes the specified path and all of its children without following symbolic links.
func (dfs *DefaultFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// DeleteBackup deletes only a backup path generated from originalPath.
func (dfs *DefaultFileSystem) DeleteBackup(backupPath string, originalPath string) error {
	fullBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return err
	}
	fullOriginalPath, err := filepath.Abs(originalPath)
	if err != nil {
		return err
	}
	fullBackupPath = filepath.Clean(fullBackupPath)
	fullOriginalPath = filepath.Clean(fullOriginalPath)
	if !isGeneratedBackupPath(fullBackupPath, fullOriginalPath+".dotfileslinker-backup") {
		return fmt.Errorf("refusing to recursively delete %q because it is not a generated backup for %q", backupPath, originalPath)
	}

	info, err := os.Lstat(fullBackupPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return os.RemoveAll(fullBackupPath)
	}
	return os.Remove(fullBackupPath)
}

func isGeneratedBackupPath(backupPath, expectedBackupPath string) bool {
	equal := func(left, right string) bool { return left == right }
	if runtime.GOOS == "windows" {
		equal = strings.EqualFold
	}
	if equal(backupPath, expectedBackupPath) {
		return true
	}
	prefix := expectedBackupPath + "."
	if len(backupPath) <= len(prefix) || !equal(backupPath[:len(prefix)], prefix) {
		return false
	}
	suffix, err := strconv.ParseUint(backupPath[len(prefix):], 10, 32)
	return err == nil && suffix > 0
}

// Move renames a file, directory, or symbolic link without following it.
func (dfs *DefaultFileSystem) Move(sourcePath string, destinationPath string) error {
	return os.Rename(sourcePath, destinationPath)
}

// CreateFileSymlink creates a symbolic link to a file at the specified path.
func (dfs *DefaultFileSystem) CreateFileSymlink(linkPath string, target string) error {
	return os.Symlink(target, linkPath)
}

// CreateDirectorySymlink creates a symbolic link to a directory at the specified path.
func (dfs *DefaultFileSystem) CreateDirectorySymlink(linkPath string, target string) error {
	return os.Symlink(target, linkPath)
}

// ReadDirectory reads all immediate children of a directory in one operation.
func (dfs *DefaultFileSystem) ReadDirectory(root string) ([]DirectoryEntry, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	result := make([]DirectoryEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, DirectoryEntry{
			Path:        filepath.Join(root, entry.Name()),
			IsDirectory: entry.IsDir(),
		})
	}
	return result, nil
}

// EnumerateFiles enumerates files that match a specific pattern in a specified directory.
func (dfs *DefaultFileSystem) EnumerateFiles(root string, pattern string, recursive bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// ディレクトリをスキップ
		if entry.IsDir() {
			// 再帰的に検索しない場合は、ルートディレクトリ以外のサブディレクトリをスキップ
			if !recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}

		// パターンに一致するファイルのみを追加
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}

		if matched {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// EnumerateDirectories enumerates immediate child directories.
func (dfs *DefaultFileSystem) EnumerateDirectories(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	directories := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, filepath.Join(root, entry.Name()))
		}
	}
	return directories, nil
}

// EnsureDirectory creates a directory at the specified path if it does not already exist.
func (dfs *DefaultFileSystem) EnsureDirectory(path string) error {
	if dfs.DirectoryExists(path) {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

// ReadAllLines reads all lines from the specified file.
func (dfs *DefaultFileSystem) ReadAllLines(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	text := string(content)
	lines := strings.Split(text, "\n")

	// Windows環境のCRLFを処理
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}

	return lines, nil
}
