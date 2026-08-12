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
	source       string
	target       string
	ensureParent bool
}

type linkDisposition uint8

const (
	linkDispositionCreate linkDisposition = iota
	linkDispositionSkip
	linkDispositionReplace
)

type validatedLinkPlanEntry struct {
	linkPlanEntry
	disposition       linkDisposition
	sourceIsDirectory bool
	validationError   error
}

type appliedLinkPlanEntry struct {
	validatedLinkPlanEntry
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

	s.logger.Info(fmt.Sprintf("Starting to link dotfiles from %s to %s", repoRoot, userHome))
	s.logger.Info(fmt.Sprintf("Using ignore file: %s", ignoreFileName))

	// Filter files in the root of the repository
	ignorePath := filepath.Join(repoRoot, ignoreFileName)
	userIgnore, err := s.loadIgnorePatterns(ignorePath)
	if err != nil {
		return err
	}
	ignoreMatcher := newIgnoreMatcher(userIgnore)
	s.logger.Verbose(fmt.Sprintf("Loaded %d user-defined ignore patterns from %s", ignoreMatcher.count(), ignorePath))
	s.logger.Verbose(fmt.Sprintf("Using %d default ignore patterns", len(defaultIgnorePatterns)))

	rootEntries, err := s.planRepositoryRoot(repoRoot, userHome, ignoreMatcher)
	if err != nil {
		return err
	}
	homeEntries, err := s.planHomeDirectory(repoRoot, userHome, ignoreMatcher)
	if err != nil {
		return err
	}
	systemEntries, err := s.planRootDirectory(repoRoot, ignoreMatcher)
	if err != nil {
		return err
	}
	plan := append(rootEntries, homeEntries...)
	plan = append(plan, systemEntries...)
	if err := validateLinkPlan(repoRoot, plan); err != nil {
		return err
	}
	validatedPlan, validationErr := s.validateLinkTargets(plan, overwrite)
	if validationErr != nil && !dryRun {
		return validationErr
	}
	if err := s.executeLinkPlan(validatedPlan, dryRun); err != nil {
		return err
	}
	if validationErr != nil {
		return validationErr
	}

	if dryRun {
		s.logger.Info("DRY RUN COMPLETED: No files were actually linked")
	} else {
		s.logger.Info("Dotfiles linking completed")
	}

	return nil
}

// planRepositoryRoot collects links for dotfiles in the repository root.
func (s *FileLinkerService) planRepositoryRoot(repoRoot string, userHome string, ignoreMatcher *ignoreMatcher) ([]linkPlanEntry, error) {
	files, err := s.fs.EnumerateFiles(repoRoot, ".*", false)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate files in repository root: %w", err)
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

	entries := make([]linkPlanEntry, 0, len(validFiles))
	for _, source := range validFiles {
		entries = append(entries, linkPlanEntry{
			source: source,
			target: filepath.Join(userHome, filepath.Base(source)),
		})
	}
	return entries, nil
}

// planHomeDirectory collects links from the HOME directory.
func (s *FileLinkerService) planHomeDirectory(repoRoot string, userHome string, ignoreMatcher *ignoreMatcher) ([]linkPlanEntry, error) {
	return s.planDirectory(repoRoot, "HOME", userHome, ignoreMatcher)
}

// planRootDirectory collects links from the ROOT directory (Linux/macOS only).
func (s *FileLinkerService) planRootDirectory(repoRoot string, ignoreMatcher *ignoreMatcher) ([]linkPlanEntry, error) {
	// Goの場合、ランタイムでOSを確認するのがより明確
	if runtime.GOOS == "windows" {
		s.logger.Info("Skipping ROOT directory processing on non-Unix platforms")
		return nil, nil
	}
	return s.planDirectory(repoRoot, "ROOT", "/", ignoreMatcher)
}

// planDirectory collects links from a structured source directory.
func (s *FileLinkerService) planDirectory(repoRoot string, srcDir string, destDir string, ignoreMatcher *ignoreMatcher) ([]linkPlanEntry, error) {
	srcPath := filepath.Join(repoRoot, srcDir)
	if !s.fs.DirectoryExists(srcPath) {
		s.logger.Info(fmt.Sprintf("%s directory not found: %s", srcDir, srcPath))
		return nil, nil
	}

	s.logger.Info(fmt.Sprintf("Processing %s directory: %s", srcDir, srcPath))
	files, ignoredFiles, err := s.collectFiles(repoRoot, srcPath, ignoreMatcher)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate files in %s: %w", srcDir, err)
	}

	// Log ignored files
	if len(ignoredFiles) > 0 {
		s.logger.Info(fmt.Sprintf("Ignoring %d files from %s directory based on ignore patterns:", len(ignoredFiles), srcDir))
		for _, file := range ignoredFiles {
			s.logger.Verbose(fmt.Sprintf("  Ignored file: %s (matched ignore pattern)", file))
		}
	}

	s.logger.Info(fmt.Sprintf("Found %d files to link from %s directory to %s", len(files), srcDir, destDir))

	entries := make([]linkPlanEntry, 0, len(files))
	for _, file := range files {
		rel, err := filepath.Rel(srcPath, file)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path: %w", err)
		}
		entries = append(entries, linkPlanEntry{
			source:       file,
			target:       filepath.Join(destDir, rel),
			ensureParent: true,
		})
	}
	return entries, nil
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

func (s *FileLinkerService) validateLinkTargets(plan []linkPlanEntry, overwrite bool) ([]validatedLinkPlanEntry, error) {
	validated := make([]validatedLinkPlanEntry, len(plan))
	var validationErrors []error
	for i, entry := range plan {
		operation := validatedLinkPlanEntry{
			linkPlanEntry:     entry,
			disposition:       linkDispositionCreate,
			sourceIsDirectory: s.fs.DirectoryExists(entry.source),
		}

		exists, err := s.fs.PathExists(entry.target)
		if err != nil {
			operation.validationError = fmt.Errorf("failed to inspect target %s: %w", entry.target, err)
		} else if exists {
			currentLinkTarget := s.fs.GetLinkTarget(entry.target)
			switch {
			case util.LinkTargetEquals(entry.target, currentLinkTarget, entry.source):
				operation.disposition = linkDispositionSkip
			case overwrite:
				operation.disposition = linkDispositionReplace
			default:
				operation.validationError = fmt.Errorf("'%s' already exists; use --force to overwrite", entry.target)
			}
		}

		if operation.validationError != nil {
			validationErrors = append(validationErrors, operation.validationError)
		}
		validated[i] = operation
	}
	return validated, errors.Join(validationErrors...)
}

func (s *FileLinkerService) executeLinkPlan(plan []validatedLinkPlanEntry, dryRun bool) error {
	if dryRun {
		for _, entry := range plan {
			s.logDryRunOperation(entry)
		}
		return nil
	}

	// Prepare every destination directory before creating the first link. In
	// particular, this prevents a ROOT permission error from occurring after
	// HOME links have already been created.
	for _, entry := range plan {
		if entry.disposition == linkDispositionSkip || !entry.ensureParent {
			continue
		}
		parent := filepath.Dir(entry.target)
		s.logger.Verbose(fmt.Sprintf("Ensuring directory exists: %s", parent))
		if err := s.fs.EnsureDirectory(parent); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", parent, err)
		}
	}

	applied := make([]appliedLinkPlanEntry, 0, len(plan))
	for _, entry := range plan {
		s.logger.Verbose(fmt.Sprintf("Linking %s to %s", entry.source, entry.target))
		operation, err := s.applyLink(entry)
		if err != nil {
			return errors.Join(err, s.rollbackLinkPlan(applied))
		}
		if operation != nil {
			applied = append(applied, *operation)
		}
	}

	var cleanupErrors []error
	for _, operation := range applied {
		if operation.backupPath == "" {
			continue
		}
		if err := s.fs.Delete(operation.backupPath); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf(
				"failed to remove replacement backup %s; the link was applied and the backup was left in place: %w",
				operation.backupPath, err))
		}
	}

	for _, operation := range applied {
		s.logger.Success(fmt.Sprintf("Creating symbolic link: %s -> %s", operation.target, operation.source))
	}
	return errors.Join(cleanupErrors...)
}

func (s *FileLinkerService) logDryRunOperation(entry validatedLinkPlanEntry) {
	s.logger.Verbose(fmt.Sprintf("Linking %s to %s", entry.source, entry.target))
	if entry.validationError != nil {
		s.logger.Error(fmt.Sprintf("[DRY-RUN] Cannot link %s to %s: %s", entry.source, entry.target, entry.validationError))
		return
	}
	if entry.disposition == linkDispositionSkip {
		s.logger.Success(fmt.Sprintf("[DRY-RUN] Would skip already linked: %s -> %s", entry.target, entry.source))
		return
	}
	if entry.disposition == linkDispositionReplace {
		s.logger.Verbose(fmt.Sprintf("[DRY-RUN] Would replace existing target: %s", entry.target))
	}
	if entry.sourceIsDirectory {
		s.logger.Success(fmt.Sprintf("[DRY-RUN] Would create directory symlink: %s -> %s", entry.target, entry.source))
	} else {
		s.logger.Success(fmt.Sprintf("[DRY-RUN] Would create file symlink: %s -> %s", entry.target, entry.source))
	}
}

// collectFiles collects linkable files without descending into ignored directories.
func (s *FileLinkerService) collectFiles(repoRoot string, sourceRoot string, ignoreMatcher *ignoreMatcher) ([]string, []string, error) {
	pendingDirectories := []string{sourceRoot}
	var files []string
	var ignoredPaths []string

	for len(pendingDirectories) > 0 {
		last := len(pendingDirectories) - 1
		currentDirectory := pendingDirectories[last]
		pendingDirectories = pendingDirectories[:last]

		directories, err := s.fs.EnumerateDirectories(currentDirectory)
		if err != nil {
			return nil, nil, err
		}
		for _, directory := range directories {
			relativePath, err := filepath.Rel(repoRoot, directory)
			if err != nil {
				return nil, nil, err
			}
			if shouldIgnoreFile(relativePath, true, ignoreMatcher) {
				ignoredPaths = append(ignoredPaths, directory)
				continue
			}
			pendingDirectories = append(pendingDirectories, directory)
		}

		currentFiles, err := s.fs.EnumerateFiles(currentDirectory, "*", false)
		if err != nil {
			return nil, nil, err
		}
		for _, file := range currentFiles {
			relativePath, err := filepath.Rel(repoRoot, file)
			if err != nil {
				return nil, nil, err
			}
			if shouldIgnoreFile(relativePath, false, ignoreMatcher) {
				ignoredPaths = append(ignoredPaths, file)
			} else {
				files = append(files, file)
			}
		}
	}

	return files, ignoredPaths, nil
}

// linkFile creates a symbolic link from the source to the target path.
func (s *FileLinkerService) linkFile(source string, target string, overwrite bool, dryRun bool) error {
	validated, validationErr := s.validateLinkTargets([]linkPlanEntry{{source: source, target: target}}, overwrite)
	if validationErr != nil {
		if dryRun {
			s.logDryRunOperation(validated[0])
		}
		return validationErr
	}
	return s.executeLinkPlan(validated, dryRun)
}

func (s *FileLinkerService) applyLink(entry validatedLinkPlanEntry) (*appliedLinkPlanEntry, error) {
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

	return &appliedLinkPlanEntry{validatedLinkPlanEntry: entry, backupPath: backupPath}, nil
}

func (s *FileLinkerService) rollbackLinkPlan(applied []appliedLinkPlanEntry) error {
	var rollbackErrors []error
	for i := len(applied) - 1; i >= 0; i-- {
		operation := applied[i]
		exists, err := s.fs.PathExists(operation.target)
		if err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to inspect destination %s during rollback: %w", operation.target, err))
			continue
		}
		if exists {
			if err := s.fs.Delete(operation.target); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to remove destination %s during rollback: %w", operation.target, err))
				continue
			}
		}
		if operation.backupPath != "" {
			if err := s.fs.Move(operation.backupPath, operation.target); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to restore destination %s during rollback: %w", operation.target, err))
			}
		}
	}
	return errors.Join(rollbackErrors...)
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

	s.logger.Verbose(fmt.Sprintf("Temporarily moving existing target: %s -> %s", target, backupPath))
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
		s.logger.Verbose(fmt.Sprintf("Ignore file not found: %s", ignoreFilePath))
		return nil, nil
	}

	lines, err := s.fs.ReadAllLines(ignoreFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ignore file %s: %w", ignoreFilePath, err)
	}

	return lines, nil
}
