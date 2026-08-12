package service

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/guitarrapc/dotfileslinker-go/internal/infrastructure"
	"github.com/guitarrapc/dotfileslinker-go/internal/util"
)

// FileLinkerService provides functionality to link dotfiles from a repository to user's home directory or system root.
type FileLinkerService struct {
	fs     infrastructure.FileSystem
	logger Logger
}

type linkPlanEntry struct {
	source            string
	target            string
	ensureParent      bool
	disposition       linkDisposition
	sourceIsDirectory bool
	validationError   error
}

type linkDisposition uint8

const (
	linkDispositionCreate linkDisposition = iota
	linkDispositionSkip
	linkDispositionReplace
)

type appliedLinkPlanEntry struct {
	linkPlanEntry
	backupPath string
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
	absoluteRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve repository root %q: %w", repoRoot, err)
	}
	absoluteUserHome, err := filepath.Abs(userHome)
	if err != nil {
		return fmt.Errorf("failed to resolve user home %q: %w", userHome, err)
	}
	repoRoot = absoluteRepoRoot
	userHome = absoluteUserHome
	if util.IsSameOrDescendant(userHome, repoRoot) {
		return fmt.Errorf("user home %q must not be the repository root or one of its descendants", userHome)
	}

	if dryRun {
		s.logger.Info("DRY RUN MODE: No files will be actually linked")
	}

	s.logger.Infof("Starting to link dotfiles from %s to %s", repoRoot, userHome)
	s.logger.Infof("Using ignore file: %s", ignoreFileName)

	// Filter files in the root of the repository
	ignorePath := filepath.Join(repoRoot, ignoreFileName)
	userIgnore, err := s.loadIgnorePatterns(ignorePath)
	if err != nil {
		return err
	}
	ignoreMatcher := newIgnoreMatcher(userIgnore)
	s.logger.Verbosef("Loaded %d user-defined ignore patterns from %s", ignoreMatcher.count(), ignorePath)
	s.logger.Verbosef("Using %d default ignore patterns", len(defaultIgnorePatterns))

	var plan []linkPlanEntry
	if err := s.appendRepositoryRootPlan(&plan, repoRoot, userHome, ignoreMatcher); err != nil {
		return err
	}
	if err := s.appendDirectoryPlan(&plan, repoRoot, "HOME", userHome, ignoreMatcher); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		s.logger.Info("Skipping ROOT directory processing on non-Unix platforms")
	} else if err := s.appendDirectoryPlan(&plan, repoRoot, "ROOT", "/", ignoreMatcher); err != nil {
		return err
	}
	if err := validateLinkPlan(repoRoot, plan); err != nil {
		return err
	}
	s.validateLinkTargets(plan, overwrite)
	if err := s.executeLinkPlan(plan, dryRun); err != nil {
		return err
	}

	if dryRun {
		s.logger.Info("DRY RUN COMPLETED: No files were actually linked")
	} else {
		s.logger.Info("Dotfiles linking completed")
	}

	return nil
}

// appendRepositoryRootPlan appends links for dotfiles in the repository root.
func (s *FileLinkerService) appendRepositoryRootPlan(plan *[]linkPlanEntry, repoRoot string, userHome string, ignoreMatcher *ignoreMatcher) error {
	children, err := s.fs.ReadDirectory(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to read repository root: %w", err)
	}
	ignoredCount := 0
	start := len(*plan)
	for _, child := range children {
		if child.IsDirectory || !strings.HasPrefix(filepath.Base(child.Path), ".") {
			continue
		}
		file := child.Path
		relPath, err := filepath.Rel(repoRoot, file)
		if err != nil {
			// If we can't get relative path, use just the filename
			relPath = filepath.Base(file)
		}
		isDir := s.fs.DirectoryExists(file)

		if shouldIgnoreFile(relPath, isDir, ignoreMatcher) {
			ignoredCount++
			s.logger.Verbosef("  Ignored file: %s (matched ignore pattern)", filepath.Base(file))
		} else {
			*plan = append(*plan, linkPlanEntry{
				source: file,
				target: filepath.Join(userHome, filepath.Base(file)),
			})
		}
	}

	if ignoredCount > 0 {
		s.logger.Infof("Ignored %d files from repository root based on ignore patterns", ignoredCount)
	}

	s.logger.Infof("Found %d files to link from repository root directory to %s", len(*plan)-start, userHome)
	return nil
}

// appendDirectoryPlan appends links from a structured source directory.
func (s *FileLinkerService) appendDirectoryPlan(plan *[]linkPlanEntry, repoRoot string, srcDir string, destDir string, ignoreMatcher *ignoreMatcher) error {
	srcPath := filepath.Join(repoRoot, srcDir)
	if !s.fs.DirectoryExists(srcPath) {
		s.logger.Infof("%s directory not found: %s", srcDir, srcPath)
		return nil
	}

	s.logger.Infof("Processing %s directory: %s", srcDir, srcPath)
	start := len(*plan)
	ignoredCount, err := s.collectLinkPlanEntries(plan, repoRoot, srcPath, destDir, ignoreMatcher)
	if err != nil {
		return fmt.Errorf("failed to enumerate files in %s: %w", srcDir, err)
	}

	if ignoredCount > 0 {
		s.logger.Infof("Ignored %d files from %s directory based on ignore patterns", ignoredCount, srcDir)
	}

	s.logger.Infof("Found %d files to link from %s directory to %s", len(*plan)-start, srcDir, destDir)
	return nil
}

func validateLinkPlan(repoRoot string, plan []linkPlanEntry) error {
	seenTargets := make(map[string]string, len(plan))
	for i, entry := range plan {
		if util.PathEquals(entry.source, entry.target) {
			return fmt.Errorf("source and destination resolve to the same path: %q", entry.source)
		}
		if util.PathsOverlap(repoRoot, entry.target) {
			return fmt.Errorf("destination %q overlaps dotfiles repository %q", entry.target, repoRoot)
		}

		key := filepath.Clean(entry.target)
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if previousSource, exists := seenTargets[key]; exists {
			return fmt.Errorf("multiple sources map to destination %q: %q and %q", entry.target, previousSource, entry.source)
		}
		for previousIndex := 0; previousIndex < i; previousIndex++ {
			previousTarget := plan[previousIndex].target
			if util.PathsOverlap(previousTarget, entry.target) {
				return fmt.Errorf("destinations %q and %q overlap", previousTarget, entry.target)
			}
		}
		seenTargets[key] = entry.source
	}
	return nil
}

func (s *FileLinkerService) validateLinkTargets(plan []linkPlanEntry, overwrite bool) {
	for i := range plan {
		operation := &plan[i]
		operation.disposition = linkDispositionCreate
		operation.sourceIsDirectory = s.fs.DirectoryExists(operation.source)

		exists, err := s.fs.PathExists(operation.target)
		if err != nil {
			operation.validationError = fmt.Errorf("failed to inspect target %s: %w", operation.target, err)
		} else if exists {
			currentLinkTarget := s.fs.GetLinkTarget(operation.target)
			switch {
			case util.LinkTargetEquals(operation.target, currentLinkTarget, operation.source):
				operation.disposition = linkDispositionSkip
			case overwrite:
				operation.disposition = linkDispositionReplace
			default:
				operation.validationError = fmt.Errorf("'%s' already exists; use --force to overwrite", operation.target)
			}
		}
	}
}

func (s *FileLinkerService) executeLinkPlan(plan []linkPlanEntry, dryRun bool) error {
	if dryRun {
		var validationErrors []error
		for _, entry := range plan {
			s.logDryRunOperation(entry)
			if entry.validationError != nil {
				validationErrors = append(validationErrors, entry.validationError)
			}
		}
		return errors.Join(validationErrors...)
	}

	applied := make([]appliedLinkPlanEntry, 0, len(plan))
	var operationErrors []error
	for _, entry := range plan {
		if entry.validationError != nil {
			s.logger.Error(fmt.Sprintf("Cannot link %s to %s: %s", entry.source, entry.target, entry.validationError))
			operationErrors = append(operationErrors, entry.validationError)
			continue
		}
		if entry.disposition != linkDispositionSkip && entry.ensureParent {
			parent := filepath.Dir(entry.target)
			s.logger.Verbosef("Ensuring directory exists: %s", parent)
			if err := s.fs.EnsureDirectory(parent); err != nil {
				operationErr := fmt.Errorf("failed to create directory %s for %s: %w", parent, entry.target, err)
				s.logger.Error(operationErr.Error())
				operationErrors = append(operationErrors, operationErr)
				continue
			}
		}
		s.logger.Verbosef("Linking %s to %s", entry.source, entry.target)
		operation, err := s.applyLink(entry)
		if err != nil {
			operationErrors = append(operationErrors, err)
			continue
		}
		if operation != nil {
			applied = append(applied, *operation)
		}
	}

	// Every entry has now reached a terminal state. Successful links are
	// committed independently, even if another entry failed. Backup cleanup is
	// post-commit work: failures are reported but never roll back links or parent
	// directories. A later run will safely skip links that already point to
	// their expected sources and retry failed entries.
	var cleanupErrors []error
	for _, operation := range applied {
		if operation.backupPath == "" {
			continue
		}
		if err := s.fs.RemoveAll(operation.backupPath); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf(
				"links are committed, but failed to remove replacement backup %s; cleanup may be incomplete: %w",
				operation.backupPath, err))
		}
	}

	for _, operation := range applied {
		s.logger.Success(fmt.Sprintf("Creating symbolic link: %s -> %s", operation.target, operation.source))
	}
	return errors.Join(errors.Join(operationErrors...), errors.Join(cleanupErrors...))
}

func (s *FileLinkerService) logDryRunOperation(entry linkPlanEntry) {
	s.logger.Verbosef("Linking %s to %s", entry.source, entry.target)
	if entry.validationError != nil {
		s.logger.Error(fmt.Sprintf("[DRY-RUN] Cannot link %s to %s: %s", entry.source, entry.target, entry.validationError))
		return
	}
	if entry.disposition == linkDispositionSkip {
		s.logger.Success(fmt.Sprintf("[DRY-RUN] Would skip already linked: %s -> %s", entry.target, entry.source))
		return
	}
	if entry.disposition == linkDispositionReplace {
		s.logger.Verbosef("[DRY-RUN] Would replace existing target: %s", entry.target)
	}
	if entry.sourceIsDirectory {
		s.logger.Success(fmt.Sprintf("[DRY-RUN] Would create directory symlink: %s -> %s", entry.target, entry.source))
	} else {
		s.logger.Success(fmt.Sprintf("[DRY-RUN] Would create file symlink: %s -> %s", entry.target, entry.source))
	}
}

// collectLinkPlanEntries collects linkable files directly into plan entries
// without descending into ignored directories.
func (s *FileLinkerService) collectLinkPlanEntries(plan *[]linkPlanEntry, repoRoot string, sourceRoot string, destinationRoot string, ignoreMatcher *ignoreMatcher) (int, error) {
	pendingDirectories := []string{sourceRoot}
	ignoredCount := 0

	for len(pendingDirectories) > 0 {
		last := len(pendingDirectories) - 1
		currentDirectory := pendingDirectories[last]
		pendingDirectories = pendingDirectories[:last]

		children, err := s.fs.ReadDirectory(currentDirectory)
		if err != nil {
			return 0, err
		}
		for _, child := range children {
			repositoryRelativePath, err := filepath.Rel(repoRoot, child.Path)
			if err != nil {
				return 0, err
			}

			if child.IsDirectory {
				if shouldIgnoreFile(repositoryRelativePath, true, ignoreMatcher) {
					ignoredCount++
					s.logger.Verbosef("  Ignored file: %s (matched ignore pattern)", child.Path)
					continue
				}
				pendingDirectories = append(pendingDirectories, child.Path)
				continue
			}
			if shouldIgnoreFile(repositoryRelativePath, false, ignoreMatcher) {
				ignoredCount++
				s.logger.Verbosef("  Ignored file: %s (matched ignore pattern)", child.Path)
				continue
			}

			destinationRelativePath, err := filepath.Rel(sourceRoot, child.Path)
			if err != nil {
				return 0, err
			}
			*plan = append(*plan, linkPlanEntry{
				source:       child.Path,
				target:       filepath.Join(destinationRoot, destinationRelativePath),
				ensureParent: true,
			})
		}
	}

	return ignoredCount, nil
}

// linkFile creates a symbolic link from the source to the target path.
func (s *FileLinkerService) linkFile(source string, target string, overwrite bool, dryRun bool) error {
	plan := []linkPlanEntry{{source: source, target: target}}
	s.validateLinkTargets(plan, overwrite)
	return s.executeLinkPlan(plan, dryRun)
}

func (s *FileLinkerService) applyLink(entry linkPlanEntry) (*appliedLinkPlanEntry, error) {
	if entry.disposition == linkDispositionSkip {
		s.logger.Success(fmt.Sprintf("Skipping already linked: %s -> %s", entry.target, entry.source))
		return nil, nil
	}

	backupPath := ""
	if entry.disposition == linkDispositionReplace {
		var err error
		backupPath, err = s.moveTargetAside(entry.target)
		if err != nil {
			return nil, err
		}
	}

	var linkErr error
	if entry.sourceIsDirectory {
		linkErr = s.fs.CreateDirectorySymlink(entry.target, entry.source)
	} else {
		linkErr = s.fs.CreateFileSymlink(entry.target, entry.source)
	}
	if linkErr != nil {
		if backupPath != "" {
			linkErr = errors.Join(linkErr, s.restoreMovedTarget(entry.target, backupPath))
		}
		s.logger.Error(fmt.Sprintf("Failed to create symlink from %s to %s: %s", entry.source, entry.target, linkErr))
		return nil, fmt.Errorf("failed to create symlink from %s to %s: %w", entry.source, entry.target, linkErr)
	}

	return &appliedLinkPlanEntry{linkPlanEntry: entry, backupPath: backupPath}, nil
}

func (s *FileLinkerService) moveTargetAside(target string) (string, error) {
	backupPath := target + ".dotfileslinker-backup"
	for suffix := 1; ; suffix++ {
		exists, err := s.fs.PathExists(backupPath)
		if err != nil {
			return "", fmt.Errorf("failed to inspect backup path for %s: %w", target, err)
		}
		if !exists {
			break
		}
		backupPath = fmt.Sprintf("%s.dotfileslinker-backup.%d", target, suffix)
	}

	s.logger.Verbosef("Temporarily moving existing target: %s -> %s", target, backupPath)
	if err := s.fs.Move(target, backupPath); err != nil {
		return "", fmt.Errorf("failed to move existing target aside: %w", err)
	}
	return backupPath, nil
}

func (s *FileLinkerService) restoreMovedTarget(target, backupPath string) error {
	var rollbackErrors []error
	exists, err := s.fs.PathExists(target)
	if err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to inspect replacement during rollback: %w", err))
	} else if exists {
		if err := s.fs.Delete(target); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to remove replacement during rollback: %w", err))
		}
	}
	if len(rollbackErrors) == 0 {
		if err := s.fs.Move(backupPath, target); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to restore original target: %w", err))
		}
	}
	return errors.Join(rollbackErrors...)
}

// shouldIgnoreFile applies Git-compatible ignore rules to a relative path.
func shouldIgnoreFile(filePath string, isDir bool, matcher *ignoreMatcher) bool {
	normalizedPath := filepath.ToSlash(filePath)
	return defaultIgnoreMatcher.ignored(normalizedPath, isDir) || matcher.ignored(normalizedPath, isDir)
}

// loadIgnorePatterns loads .gitignore-compatible pattern lines in source order.
// A missing ignore file is optional; all other inspection and read errors are fatal.
func (s *FileLinkerService) loadIgnorePatterns(ignoreFilePath string) ([]string, error) {
	exists, err := s.fs.PathExists(ignoreFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect ignore file %s: %w", ignoreFilePath, err)
	}
	if !exists {
		s.logger.Verbosef("Ignore file not found: %s", ignoreFilePath)
		return nil, nil
	}

	lines, err := s.fs.ReadAllLines(ignoreFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ignore file %s: %w", ignoreFilePath, err)
	}

	return lines, nil
}
