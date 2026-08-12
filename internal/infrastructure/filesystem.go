package infrastructure

// DirectoryEntry describes one immediate child returned by ReadDirectory.
type DirectoryEntry struct {
	Path        string
	IsDirectory bool
}

// FileSystem provides an abstraction for file system operations to support testing and platform-specific behavior.
type FileSystem interface {
	// FileExists determines whether the specified file exists.
	FileExists(path string) bool

	// DirectoryExists determines whether the specified directory exists.
	DirectoryExists(path string) bool

	// PathExists determines whether a directory entry exists without following
	// symbolic links, so dangling links are reported as existing.
	PathExists(path string) (bool, error)

	// GetLinkTarget gets the target of a symbolic link at the specified path.
	// Returns empty string if the path is not a symbolic link.
	GetLinkTarget(path string) string

	// Delete deletes the specified file or empty directory.
	Delete(path string) error

	// DeleteBackup deletes a generated replacement backup. Implementations must
	// verify that backupPath belongs to originalPath before recursive deletion.
	DeleteBackup(backupPath string, originalPath string) error

	// Move renames a file, directory, or symbolic link without following it.
	Move(sourcePath string, destinationPath string) error

	// CreateFileSymlink creates a symbolic link to a file at the specified path.
	CreateFileSymlink(linkPath string, target string) error

	// CreateDirectorySymlink creates a symbolic link to a directory at the specified path.
	CreateDirectorySymlink(linkPath string, target string) error

	// ReadDirectory reads all immediate children of a directory in one operation.
	ReadDirectory(root string) ([]DirectoryEntry, error)

	// EnumerateFiles enumerates files that match a pattern.
	EnumerateFiles(root string, pattern string, recursive bool) ([]string, error)

	// EnumerateDirectories enumerates immediate child directories.
	EnumerateDirectories(root string) ([]string, error)

	// EnsureDirectory creates a directory at the specified path if it does not already exist.
	EnsureDirectory(path string) error

	// ReadAllLines reads all lines from the specified file.
	ReadAllLines(path string) ([]string, error)
}
