package infrastructure

import (
	"errors"
	"path/filepath"
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
	return []string{}, nil
}

// EnsureDirectory creates a directory if it doesn't exist
func (m *MockFileSystem) EnsureDirectory(path string) error {
	m.OperationLog = append(m.OperationLog, "EnsureDirectory: "+path)
	if err, exists := m.ErrorResponses["EnsureDirectory:"+path]; exists {
		return err
	}

	m.Directories[path] = true
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
	// When adding a file, ensure its directory exists
	dir := filepath.Dir(path)
	m.Directories[dir] = true
}

// AddDirectory adds a directory to the mock filesystem
func (m *MockFileSystem) AddDirectory(path string) {
	m.Directories[path] = true
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
