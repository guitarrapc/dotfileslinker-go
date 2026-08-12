package infrastructure

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
)

// MockFileSystem implements FileSystem interface for testing purposes
type MockFileSystem struct {
	Files            map[string]string   // Map of path to file content
	Directories      map[string]bool     // Map of existing directories
	SymLinks         map[string]string   // Map of symlink paths to targets
	FileEnumerations map[string][]string // Map of path pattern to enumerated files
	ErrorResponses   map[string]error    // Map of operations to errors
	OperationLog     []string            // Log of performed operations
}

// NewMockFileSystem creates a new instance of MockFileSystem
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files:            make(map[string]string),
		Directories:      make(map[string]bool),
		SymLinks:         make(map[string]string),
		FileEnumerations: make(map[string][]string),
		ErrorResponses:   make(map[string]error),
	}
}

// FileExists checks if a file exists
func (m *MockFileSystem) FileExists(path string) bool {
	m.OperationLog = append(m.OperationLog, "FileExists: "+path)
	_, exists := m.Files[path]
	return exists
}

// DirectoryExists checks if a directory exists
func (m *MockFileSystem) DirectoryExists(path string) bool {
	m.OperationLog = append(m.OperationLog, "DirectoryExists: "+path)
	_, exists := m.Directories[path]
	return exists
}

// PathExists checks for a file, directory, or symbolic-link entry.
func (m *MockFileSystem) PathExists(path string) (bool, error) {
	m.OperationLog = append(m.OperationLog, "PathExists: "+path)
	if err, exists := m.ErrorResponses["PathExists:"+path]; exists {
		return false, err
	}
	if _, exists := m.Files[path]; exists {
		return true, nil
	}
	if _, exists := m.Directories[path]; exists {
		return true, nil
	}
	_, exists := m.SymLinks[path]
	return exists, nil
}

// GetLinkTarget gets the target of a symbolic link
func (m *MockFileSystem) GetLinkTarget(path string) string {
	m.OperationLog = append(m.OperationLog, "GetLinkTarget: "+path)
	target, exists := m.SymLinks[path]
	if exists {
		return target
	}
	return ""
}

// Delete removes a file or directory
func (m *MockFileSystem) Delete(path string) error {
	m.OperationLog = append(m.OperationLog, "Delete: "+path)
	if err, exists := m.ErrorResponses["Delete:"+path]; exists {
		return err
	}

	delete(m.Files, path)
	delete(m.Directories, path)
	delete(m.SymLinks, path)
	return nil
}

// RemoveAll removes a path and every mock entry below it.
func (m *MockFileSystem) RemoveAll(path string) error {
	m.OperationLog = append(m.OperationLog, "RemoveAll: "+path)
	if err, exists := m.ErrorResponses["RemoveAll:"+path]; exists {
		return err
	}

	for file := range m.Files {
		if isSameOrDescendant(file, path) {
			delete(m.Files, file)
		}
	}
	for directory := range m.Directories {
		if isSameOrDescendant(directory, path) {
			delete(m.Directories, directory)
		}
	}
	for link := range m.SymLinks {
		if isSameOrDescendant(link, path) {
			delete(m.SymLinks, link)
		}
	}
	return nil
}

// Move renames a file, directory, or symbolic link.
func (m *MockFileSystem) Move(sourcePath string, destinationPath string) error {
	operation := "Move: " + sourcePath + " -> " + destinationPath
	m.OperationLog = append(m.OperationLog, operation)
	if err, exists := m.ErrorResponses["Move:"+sourcePath+"->"+destinationPath]; exists {
		return err
	}

	if content, exists := m.Files[sourcePath]; exists {
		delete(m.Files, sourcePath)
		m.Files[destinationPath] = content
		return nil
	}
	if _, exists := m.Directories[sourcePath]; exists {
		delete(m.Directories, sourcePath)
		m.Directories[destinationPath] = true
		return nil
	}
	if target, exists := m.SymLinks[sourcePath]; exists {
		delete(m.SymLinks, sourcePath)
		m.SymLinks[destinationPath] = target
		return nil
	}
	return errors.New("source path not found")
}

// CreateFileSymlink creates a symbolic link to a file
func (m *MockFileSystem) CreateFileSymlink(linkPath string, target string) error {
	m.OperationLog = append(m.OperationLog, "CreateFileSymlink: "+linkPath+" -> "+target)
	if err, exists := m.ErrorResponses["CreateFileSymlink:"+linkPath]; exists {
		return err
	}

	m.SymLinks[linkPath] = target
	return nil
}

// CreateDirectorySymlink creates a symbolic link to a directory
func (m *MockFileSystem) CreateDirectorySymlink(linkPath string, target string) error {
	m.OperationLog = append(m.OperationLog, "CreateDirectorySymlink: "+linkPath+" -> "+target)
	if err, exists := m.ErrorResponses["CreateDirectorySymlink:"+linkPath]; exists {
		return err
	}

	m.SymLinks[linkPath] = target
	return nil
}

// ReadDirectory reads all immediate children from the mock filesystem.
func (m *MockFileSystem) ReadDirectory(root string) ([]DirectoryEntry, error) {
	key := "ReadDirectory:" + root
	m.OperationLog = append(m.OperationLog, key)
	if err, exists := m.ErrorResponses[key]; exists {
		return nil, err
	}

	root = filepath.Clean(root)
	seen := make(map[string]struct{})
	var entries []DirectoryEntry
	appendEntry := func(path string, isDirectory bool) {
		path = filepath.Clean(path)
		if filepath.Dir(path) != root {
			return
		}
		if _, exists := seen[path]; exists {
			return
		}
		seen[path] = struct{}{}
		entries = append(entries, DirectoryEntry{Path: path, IsDirectory: isDirectory})
	}

	for directory := range m.Directories {
		appendEntry(directory, true)
	}
	for file := range m.Files {
		appendEntry(file, false)
	}
	for link := range m.SymLinks {
		appendEntry(link, false)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries, nil
}

// EnumerateFiles lists files matching a pattern
func (m *MockFileSystem) EnumerateFiles(root string, pattern string, recursive bool) ([]string, error) {
	key := "EnumerateFiles:" + root + ":" + pattern + ":" + getBoolStr(recursive)
	m.OperationLog = append(m.OperationLog, key)
	if err, exists := m.ErrorResponses[key]; exists {
		return nil, err
	}

	files, exists := m.FileEnumerations[root+":"+pattern+":"+getBoolStr(recursive)]
	if exists {
		return files, nil
	}

	root = filepath.Clean(root)
	result := make([]string, 0)
	for file := range m.Files {
		relative, err := filepath.Rel(root, file)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		if !recursive && filepath.Dir(relative) != "." {
			continue
		}
		matched, err := filepath.Match(pattern, filepath.Base(file))
		if err != nil {
			return nil, err
		}
		if matched {
			result = append(result, file)
		}
	}
	return result, nil
}

// EnumerateDirectories lists immediate child directories.
func (m *MockFileSystem) EnumerateDirectories(root string) ([]string, error) {
	key := "EnumerateDirectories:" + root
	m.OperationLog = append(m.OperationLog, key)
	if err, exists := m.ErrorResponses[key]; exists {
		return nil, err
	}

	root = filepath.Clean(root)
	directories := make([]string, 0)
	for directory := range m.Directories {
		if filepath.Dir(filepath.Clean(directory)) == root {
			directories = append(directories, directory)
		}
	}
	return directories, nil
}

// EnsureDirectory creates a directory if it doesn't exist
func (m *MockFileSystem) EnsureDirectory(path string) error {
	m.OperationLog = append(m.OperationLog, "EnsureDirectory: "+path)
	if err, exists := m.ErrorResponses["EnsureDirectory:"+path]; exists {
		return err
	}

	for directory := filepath.Clean(path); ; directory = filepath.Dir(directory) {
		m.Directories[directory] = true
		if parent := filepath.Dir(directory); parent == directory {
			break
		}
	}
	return nil
}

// ReadAllLines reads all lines from a file
func (m *MockFileSystem) ReadAllLines(path string) ([]string, error) {
	m.OperationLog = append(m.OperationLog, "ReadAllLines: "+path)
	if err, exists := m.ErrorResponses["ReadAllLines:"+path]; exists {
		return nil, err
	}

	content, exists := m.Files[path]
	if !exists {
		return nil, errors.New("file not found")
	}

	return strings.Split(content, "\n"), nil
}

// AddFile adds a file to the mock filesystem
func (m *MockFileSystem) AddFile(path string, content string) {
	m.Files[path] = content
	for directory := filepath.Dir(path); ; directory = filepath.Dir(directory) {
		m.Directories[directory] = true
		next := filepath.Dir(directory)
		if next == directory {
			break
		}
	}
}

// AddDirectory adds a directory to the mock filesystem
func (m *MockFileSystem) AddDirectory(path string) {
	for directory := filepath.Clean(path); ; directory = filepath.Dir(directory) {
		m.Directories[directory] = true
		if parent := filepath.Dir(directory); parent == directory {
			break
		}
	}
}

// SetupFileEnumeration configures file enumeration results
func (m *MockFileSystem) SetupFileEnumeration(root string, pattern string, recursive bool, files []string) {
	key := root + ":" + pattern + ":" + getBoolStr(recursive)
	m.FileEnumerations[key] = files
}

// SetErrorForOperation configures an error for a specific operation
func (m *MockFileSystem) SetErrorForOperation(operation string, err error) {
	m.ErrorResponses[operation] = err
}

// getBoolStr converts a boolean to a string
func getBoolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func isSameOrDescendant(path, directory string) bool {
	relative, err := filepath.Rel(directory, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
