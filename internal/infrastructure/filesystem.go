package infrastructure

// FileSystem provides an abstraction for file system operations to support testing and platform-specific behavior.
type FileSystem interface {
	// FileExists determines whether the specified file exists.
	FileExists(path string) bool

	// DirectoryExists determines whether the specified directory exists.
	DirectoryExists(path string) bool

	// GetLinkTarget gets the target of a symbolic link at the specified path.
	// Returns empty string if the path is not a symbolic link.
	GetLinkTarget(path string) string

	// Delete deletes the specified file or empty directory.
	Delete(path string) error

	// CreateFileSymlink creates a symbolic link to a file at the specified path.
	CreateFileSymlink(linkPath string, target string) error

	// CreateDirectorySymlink creates a symbolic link to a directory at the specified path.
	CreateDirectorySymlink(linkPath string, target string) error

	// EnumerateFiles enumerates files that match a specific pattern in a specified directory.
	EnumerateFiles(root string, pattern string, recursive bool) ([]string, error)

	// EnsureDirectory creates a directory at the specified path if it does not already exist.
	EnsureDirectory(path string) error

	// ReadAllLines reads all lines from the specified file.
	ReadAllLines(path string) ([]string, error)
}
