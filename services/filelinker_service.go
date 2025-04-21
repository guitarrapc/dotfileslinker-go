package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guitarrapc/dotfileslinker-go/infrastructure"
	"github.com/guitarrapc/dotfileslinker-go/utils"
)

// FileLinkerService provides functionality to link dotfiles from a repository to user's home directory or system root.
type FileLinkerService struct {
	fs     infrastructure.FileSystem
	logger Logger
}

// NewFileLinkerService creates a new instance of FileLinkerService.
func NewFileLinkerService(fs infrastructure.FileSystem, logger Logger) *FileLinkerService {
	if logger == nil {
		logger = NewNullLogger()
	}
	return &FileLinkerService{
		fs:     fs,
		logger: logger,
	}
}

// LinkDotfiles links dotfiles from the specified repository to the user's home directory or system root.
// repoRoot: The root directory of the dotfiles repository.
// userHome: The user's home directory path.
// ignoreFileName: The name of the ignore file containing patterns to exclude.
// overwrite: Whether to overwrite existing files or directories.
func (s *FileLinkerService) LinkDotfiles(repoRoot string, userHome string, ignoreFileName string, overwrite bool) error {
	s.logger.Info(fmt.Sprintf("Starting to link dotfiles from %s to %s", repoRoot, userHome))
	s.logger.Info(fmt.Sprintf("Using ignore file: %s", ignoreFileName))

	// Filter files in the root of the repository
	ignorePath := filepath.Join(repoRoot, ignoreFileName)
	ignore := s.loadIgnoreList(ignorePath)
	s.logger.Verbose(fmt.Sprintf("Loaded %d ignore patterns from %s", len(ignore), ignorePath))

	// Process each directory
	if err := s.processRepositoryRoot(repoRoot, userHome, ignore, overwrite); err != nil {
		return err
	}

	if err := s.processHomeDirectory(repoRoot, userHome, overwrite); err != nil {
		return err
	}

	if err := s.processRootDirectory(repoRoot, overwrite); err != nil {
		return err
	}

	s.logger.Info("Dotfiles linking completed")
	return nil
}

// processRepositoryRoot processes and links files in the repository root.
func (s *FileLinkerService) processRepositoryRoot(repoRoot string, userHome string, ignore map[string]bool, overwrite bool) error {
	files, err := s.fs.EnumerateFiles(repoRoot, ".*", false)
	if err != nil {
		return fmt.Errorf("failed to enumerate files in repository root: %w", err)
	}

	var validFiles []string
	for _, file := range files {
		baseName := filepath.Base(file)
		if _, exists := ignore[baseName]; !exists {
			validFiles = append(validFiles, file)
		}
	}

	s.logger.Info(fmt.Sprintf("Found %d files to link from repository root directory to %s", len(validFiles), userHome))

	for _, src := range validFiles {
		dst := filepath.Join(userHome, filepath.Base(src))
		s.logger.Verbose(fmt.Sprintf("Linking %s to %s", src, dst))
		if err := s.linkFile(src, dst, overwrite); err != nil {
			return err
		}
	}

	return nil
}

// processHomeDirectory processes and links files in the HOME directory.
func (s *FileLinkerService) processHomeDirectory(repoRoot string, userHome string, overwrite bool) error {
	return s.processDirectory(repoRoot, "HOME", userHome, overwrite)
}

// processRootDirectory processes and links files in the ROOT directory (Linux/macOS only).
func (s *FileLinkerService) processRootDirectory(repoRoot string, overwrite bool) error {
	// Goの場合、ランタイムでOSを確認するのがより明確
	if os.Getenv("OS") == "Windows_NT" {
		s.logger.Info("Skipping ROOT directory processing on non-Unix platforms")
		return nil
	}
	return s.processDirectory(repoRoot, "ROOT", "/", overwrite)
}

// processDirectory processes and links files in the specified directory.
func (s *FileLinkerService) processDirectory(repoRoot string, srcDir string, destDir string, overwrite bool) error {
	srcPath := filepath.Join(repoRoot, srcDir)
	if !s.fs.DirectoryExists(srcPath) {
		s.logger.Info(fmt.Sprintf("%s directory not found: %s", srcDir, srcPath))
		return nil
	}

	s.logger.Info(fmt.Sprintf("Processing %s directory: %s", srcDir, srcPath))
	files, err := s.fs.EnumerateFiles(srcPath, "*", true)
	if err != nil {
		return fmt.Errorf("failed to enumerate files in %s: %w", srcDir, err)
	}

	s.logger.Info(fmt.Sprintf("Found %d files to link from %s directory to %s", len(files), srcDir, destDir))

	for _, file := range files {
		rel, err := filepath.Rel(srcPath, file)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		dst := filepath.Join(destDir, rel)

		dstDir := filepath.Dir(dst)
		s.logger.Verbose(fmt.Sprintf("Ensuring directory exists: %s", dstDir))
		if err := s.fs.EnsureDirectory(dstDir); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		s.logger.Verbose(fmt.Sprintf("Linking %s to %s", file, dst))
		if err := s.linkFile(file, dst, overwrite); err != nil {
			return err
		}
	}

	return nil
}

// linkFile creates a symbolic link from the source to the target path.
func (s *FileLinkerService) linkFile(source string, target string, overwrite bool) error {
	fileExists := s.fs.FileExists(target)
	dirExists := s.fs.DirectoryExists(target)
	exists := fileExists || dirExists

	if exists {
		currentLinkTarget := s.fs.GetLinkTarget(target)

		// If the target is a symlink and points to the same file, do nothing
		if currentLinkTarget != "" && utils.PathEquals(currentLinkTarget, source) {
			s.logger.Success(fmt.Sprintf("Skipping already linked: %s -> %s", target, source))
			return nil
		}

		if !overwrite {
			s.logger.Verbose(fmt.Sprintf("Target %s exists and overwrite=false, aborting", target))
			return fmt.Errorf("'%s' already exists; use --force=y to overwrite", target)
		}

		s.logger.Verbose(fmt.Sprintf("Deleting existing target: %s", target))
		if err := s.fs.Delete(target); err != nil {
			return fmt.Errorf("failed to delete existing target: %w", err)
		}
	}

	// Create the link
	var err error
	if s.fs.DirectoryExists(source) {
		s.logger.Success(fmt.Sprintf("Creating directory symlink: %s -> %s", target, source))
		err = s.fs.CreateDirectorySymlink(target, source)
	} else {
		s.logger.Success(fmt.Sprintf("Creating file symlink: %s -> %s", target, source))
		err = s.fs.CreateFileSymlink(target, source)
	}

	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to create symlink from %s to %s: %s", source, target, err))
		return err
	}

	return nil
}

// loadIgnoreList loads the ignore list from the specified file.
func (s *FileLinkerService) loadIgnoreList(ignoreFilePath string) map[string]bool {
	ignore := make(map[string]bool)

	if !s.fs.FileExists(ignoreFilePath) {
		s.logger.Verbose(fmt.Sprintf("Ignore file not found: %s", ignoreFilePath))
		return ignore
	}

	lines, err := s.fs.ReadAllLines(ignoreFilePath)
	if err != nil {
		s.logger.Verbose(fmt.Sprintf("Failed to read ignore file: %s", err))
		return ignore
	}
	s.logger.Verbose(fmt.Sprintf("Loaded %d lines from ignore file", len(lines)))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			ignore[line] = true
		}
	}

	return ignore
}
