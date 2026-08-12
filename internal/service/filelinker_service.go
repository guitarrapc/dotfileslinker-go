package service

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/guitarrapc/dotfileslinker-go/internal/infrastructure"
	"github.com/guitarrapc/dotfileslinker-go/internal/util"
)

// FileLinkerService provides functionality to link dotfiles from a repository to user's home directory or system root.
type FileLinkerService struct {
	fs     infrastructure.FileSystem
	logger Logger
}

// defaultIgnorePatterns contains default patterns to ignore in all directories, common for all platforms
var defaultIgnorePatterns = []string{
	// Common OS specific files
	".DS_Store",         // macOS
	"._.DS_Store",       // macOS
	"Thumbs.db",         // Windows
	"Desktop.ini",       // Windows
	"ehthumbs.db",       // Windows
	"ehthumbs_vista.db", // Windows

	// Common backup/temporary files
	"*~",     // Linux/Unix backup files
	".*.swp", // Vim swap files
	".*.swo", // Vim swap files
	"*.bak",  // Backup files
	"*.tmp",  // Temporary files

	// Version control system folders
	".git",
	".svn",
	".hg",
}

var defaultIgnoreMatcher = newIgnoreMatcher(defaultIgnorePatterns)

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
// dryRun: If true, only shows what would be done without actually creating links.
func (s *FileLinkerService) LinkDotfiles(repoRoot string, userHome string, ignoreFileName string, overwrite bool, dryRun bool) error {
	if dryRun {
		s.logger.Info("DRY RUN MODE: No files will be actually linked")
	}

	s.logger.Info(fmt.Sprintf("Starting to link dotfiles from %s to %s", repoRoot, userHome))
	s.logger.Info(fmt.Sprintf("Using ignore file: %s", ignoreFileName))

	// Filter files in the root of the repository
	ignorePath := filepath.Join(repoRoot, ignoreFileName)
	userIgnore := s.loadIgnorePatterns(ignorePath)
	ignoreMatcher := newIgnoreMatcher(userIgnore)
	s.logger.Verbose(fmt.Sprintf("Loaded %d user-defined ignore patterns from %s", ignoreMatcher.count(), ignorePath))
	s.logger.Verbose(fmt.Sprintf("Using %d default ignore patterns", len(defaultIgnorePatterns)))

	// Process each directory
	if err := s.processRepositoryRoot(repoRoot, userHome, ignoreMatcher, overwrite, dryRun); err != nil {
		return err
	}

	if err := s.processHomeDirectory(repoRoot, userHome, ignoreMatcher, overwrite, dryRun); err != nil {
		return err
	}

	if err := s.processRootDirectory(repoRoot, ignoreMatcher, overwrite, dryRun); err != nil {
		return err
	}

	if dryRun {
		s.logger.Info("DRY RUN COMPLETED: No files were actually linked")
	} else {
		s.logger.Info("Dotfiles linking completed")
	}

	return nil
}

// processRepositoryRoot processes and links files in the repository root.
func (s *FileLinkerService) processRepositoryRoot(repoRoot string, userHome string, ignoreMatcher *ignoreMatcher, overwrite bool, dryRun bool) error {
	files, err := s.fs.EnumerateFiles(repoRoot, ".*", false)
	if err != nil {
		return fmt.Errorf("failed to enumerate files in repository root: %w", err)
	}
	var validFiles []string
	var ignoredFiles []string
	for _, file := range files {
		relPath, err := filepath.Rel(repoRoot, file)
		if err != nil {
			// If we can't get relative path, use just the filename
			relPath = filepath.Base(file)
		}
		isDir := s.fs.DirectoryExists(file)

		if shouldIgnoreFile(relPath, isDir, ignoreMatcher) {
			ignoredFiles = append(ignoredFiles, file)
		} else {
			validFiles = append(validFiles, file)
		}
	}

	// Log ignored files
	if len(ignoredFiles) > 0 {
		s.logger.Info(fmt.Sprintf("Ignoring %d files from repository root based on ignore patterns:", len(ignoredFiles)))
		for _, file := range ignoredFiles {
			s.logger.Verbose(fmt.Sprintf("  Ignored file: %s (matched ignore pattern)", filepath.Base(file)))
		}
	}

	s.logger.Info(fmt.Sprintf("Found %d files to link from repository root directory to %s", len(validFiles), userHome))

	for _, src := range validFiles {
		dst := filepath.Join(userHome, filepath.Base(src))
		s.logger.Verbose(fmt.Sprintf("Linking %s to %s", src, dst))
		if err := s.linkFile(src, dst, overwrite, dryRun); err != nil {
			return err
		}
	}

	return nil
}

// processHomeDirectory processes and links files in the HOME directory.
func (s *FileLinkerService) processHomeDirectory(repoRoot string, userHome string, ignoreMatcher *ignoreMatcher, overwrite bool, dryRun bool) error {
	return s.processDirectory(repoRoot, "HOME", userHome, ignoreMatcher, overwrite, dryRun)
}

// processRootDirectory processes and links files in the ROOT directory (Linux/macOS only).
func (s *FileLinkerService) processRootDirectory(repoRoot string, ignoreMatcher *ignoreMatcher, overwrite bool, dryRun bool) error {
	// Goの場合、ランタイムでOSを確認するのがより明確
	if runtime.GOOS == "windows" {
		s.logger.Info("Skipping ROOT directory processing on non-Unix platforms")
		return nil
	}
	return s.processDirectory(repoRoot, "ROOT", "/", ignoreMatcher, overwrite, dryRun)
}

// processDirectory processes and links files in the specified directory.
func (s *FileLinkerService) processDirectory(repoRoot string, srcDir string, destDir string, ignoreMatcher *ignoreMatcher, overwrite bool, dryRun bool) error {
	srcPath := filepath.Join(repoRoot, srcDir)
	if !s.fs.DirectoryExists(srcPath) {
		s.logger.Info(fmt.Sprintf("%s directory not found: %s", srcDir, srcPath))
		return nil
	}

	s.logger.Info(fmt.Sprintf("Processing %s directory: %s", srcDir, srcPath))
	allFiles, err := s.fs.EnumerateFiles(srcPath, "*", true)
	if err != nil {
		return fmt.Errorf("failed to enumerate files in %s: %w", srcDir, err)
	}

	// Filter files based on ignore patterns
	var files []string
	var ignoredFiles []string
	for _, file := range allFiles {
		relPath, err := filepath.Rel(repoRoot, file)
		if err != nil {
			// Preserve the source directory prefix used by root-relative patterns.
			relPath = filepath.Join(srcDir, filepath.Base(file))
		}
		isDir := s.fs.DirectoryExists(file)

		if shouldIgnoreFile(relPath, isDir, ignoreMatcher) {
			ignoredFiles = append(ignoredFiles, file)
		} else {
			files = append(files, file)
		}
	}

	// Log ignored files
	if len(ignoredFiles) > 0 {
		s.logger.Info(fmt.Sprintf("Ignoring %d files from %s directory based on ignore patterns:", len(ignoredFiles), srcDir))
		for _, file := range ignoredFiles {
			s.logger.Verbose(fmt.Sprintf("  Ignored file: %s (matched ignore pattern)", file))
		}
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

		// Only actually create the directory if not in dry-run mode
		if !dryRun {
			if err := s.fs.EnsureDirectory(dstDir); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		}

		s.logger.Verbose(fmt.Sprintf("Linking %s to %s", file, dst))
		if err := s.linkFile(file, dst, overwrite, dryRun); err != nil {
			return err
		}
	}

	return nil
}

// linkFile creates a symbolic link from the source to the target path.
func (s *FileLinkerService) linkFile(source string, target string, overwrite bool, dryRun bool) error {
	fileExists := s.fs.FileExists(target)
	dirExists := s.fs.DirectoryExists(target)
	exists := fileExists || dirExists

	if exists {
		currentLinkTarget := s.fs.GetLinkTarget(target)

		// If the target is a symlink and points to the same file, do nothing
		if currentLinkTarget != "" && util.PathEquals(currentLinkTarget, source) {
			if dryRun {
				s.logger.Success(fmt.Sprintf("[DRY-RUN] Would skip already linked: %s -> %s", target, source))
			} else {
				s.logger.Success(fmt.Sprintf("Skipping already linked: %s -> %s", target, source))
			}
			return nil
		}

		if !overwrite {
			s.logger.Verbose(fmt.Sprintf("Target %s exists and overwrite=false, aborting", target))
			return fmt.Errorf("'%s' already exists; use --force to overwrite", target)
		}

		if dryRun {
			s.logger.Verbose(fmt.Sprintf("[DRY-RUN] Would delete existing target: %s", target))
		} else {
			s.logger.Verbose(fmt.Sprintf("Deleting existing target: %s", target))
			if err := s.fs.Delete(target); err != nil {
				return fmt.Errorf("failed to delete existing target: %w", err)
			}
		}
	}

	// Create the link (or just log what would happen in dry-run mode)
	var err error
	if s.fs.DirectoryExists(source) {
		if dryRun {
			s.logger.Success(fmt.Sprintf("[DRY-RUN] Would create directory symlink: %s -> %s", target, source))
			return nil
		} else {
			s.logger.Success(fmt.Sprintf("Creating directory symlink: %s -> %s", target, source))
			err = s.fs.CreateDirectorySymlink(target, source)
		}
	} else {
		if dryRun {
			s.logger.Success(fmt.Sprintf("[DRY-RUN] Would create file symlink: %s -> %s", target, source))
			return nil
		} else {
			s.logger.Success(fmt.Sprintf("Creating file symlink: %s -> %s", target, source))
			err = s.fs.CreateFileSymlink(target, source)
		}
	}

	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to create symlink from %s to %s: %s", source, target, err))
		return err
	}

	return nil
}

// shouldIgnoreFile applies Git-compatible ignore rules to a relative path.
func shouldIgnoreFile(filePath string, isDir bool, matcher *ignoreMatcher) bool {
	normalizedPath := filepath.ToSlash(filePath)
	return defaultIgnoreMatcher.ignored(normalizedPath, isDir) || matcher.ignored(normalizedPath, isDir)
}

// loadIgnorePatterns loads .gitignore-compatible pattern lines in source order.
func (s *FileLinkerService) loadIgnorePatterns(ignoreFilePath string) []string {
	if !s.fs.FileExists(ignoreFilePath) {
		s.logger.Verbose(fmt.Sprintf("Ignore file not found: %s", ignoreFilePath))
		return nil
	}

	lines, err := s.fs.ReadAllLines(ignoreFilePath)
	if err != nil {
		s.logger.Verbose(fmt.Sprintf("Failed to read ignore file: %s", err))
		return nil
	}

	return lines
}
