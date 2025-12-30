package infrastructure

import (
	"os"
	"path/filepath"
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

// CreateFileSymlink creates a symbolic link to a file at the specified path.
func (dfs *DefaultFileSystem) CreateFileSymlink(linkPath string, target string) error {
	return os.Symlink(target, linkPath)
}

// CreateDirectorySymlink creates a symbolic link to a directory at the specified path.
func (dfs *DefaultFileSystem) CreateDirectorySymlink(linkPath string, target string) error {
	return os.Symlink(target, linkPath)
}

// EnumerateFiles enumerates files that match a specific pattern in a specified directory.
func (dfs *DefaultFileSystem) EnumerateFiles(root string, pattern string, recursive bool) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// ディレクトリをスキップ
		if info.IsDir() {
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
