package service

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/guitarrapc/dotfileslinker-go/internal/infrastructure"
)

type linkPlanBuilder struct {
	fs     infrastructure.FileSystem
	logger Logger
}

var defaultIgnorePatterns = []string{
	".DS_Store", "._.DS_Store", "Thumbs.db", "Desktop.ini", "ehthumbs.db", "ehthumbs_vista.db",
	"*~", ".*.swp", ".*.swo", "*.bak", "*.tmp",
	".git", ".svn", ".hg",
}

var defaultIgnoreMatcher = newIgnoreMatcher(defaultIgnorePatterns)

func newLinkPlanBuilder(fs infrastructure.FileSystem, logger Logger) *linkPlanBuilder {
	return &linkPlanBuilder{fs: fs, logger: logger}
}

func (b *linkPlanBuilder) build(repoRoot, userHome, ignoreFileName string) (linkPlan, error) {
	ignorePath := filepath.Join(repoRoot, ignoreFileName)
	userIgnore, err := b.loadIgnorePatterns(ignorePath)
	if err != nil {
		return nil, err
	}
	directoryPatterns, err := b.loadPatterns(filepath.Join(repoRoot, "dotfiles_link_dirs"), "directory-link")
	if err != nil {
		return nil, err
	}
	directoryLinks := newIgnoreMatcher(directoryPatterns)
	b.logger.Verbosef("Loaded %d directory-link patterns from %s", directoryLinks.count(), filepath.Join(repoRoot, "dotfiles_link_dirs"))
	ignoreMatcher := newIgnoreMatcher(userIgnore)
	b.logger.Verbosef("Loaded %d user-defined ignore patterns from %s", ignoreMatcher.count(), ignorePath)
	b.logger.Verbosef("Using %d default ignore patterns", len(defaultIgnorePatterns))

	var plan linkPlan
	if err := b.appendRepositoryRootPlan(&plan, repoRoot, userHome, ignoreMatcher); err != nil {
		return nil, err
	}
	if err := b.appendDirectoryPlan(&plan, repoRoot, "HOME", userHome, ignoreMatcher, directoryLinks); err != nil {
		return nil, err
	}
	if runtime.GOOS == "windows" {
		b.logger.Info("Skipping ROOT directory processing on non-Unix platforms")
	} else if err := b.appendDirectoryPlan(&plan, repoRoot, "ROOT", "/", ignoreMatcher, directoryLinks); err != nil {
		return nil, err
	}
	if err := plan.validate(repoRoot); err != nil {
		return nil, err
	}
	return plan, nil
}

func (b *linkPlanBuilder) appendRepositoryRootPlan(plan *linkPlan, repoRoot, userHome string, matcher *ignoreMatcher) error {
	children, err := b.fs.ReadDirectory(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to read repository root: %w", err)
	}
	ignoredCount, start := 0, len(*plan)
	for _, child := range children {
		if child.IsDirectory || !strings.HasPrefix(filepath.Base(child.Path), ".") {
			continue
		}
		relPath, err := filepath.Rel(repoRoot, child.Path)
		if err != nil {
			relPath = filepath.Base(child.Path)
		}
		if shouldIgnoreFile(relPath, false, matcher) {
			ignoredCount++
			b.logger.Verbosef("  Ignored file: %s (matched ignore pattern)", filepath.Base(child.Path))
			continue
		}
		*plan = append(*plan, linkPlanEntry{source: child.Path, target: filepath.Join(userHome, filepath.Base(child.Path))})
	}
	if ignoredCount > 0 {
		b.logger.Infof("Ignored %d files from repository root based on ignore patterns", ignoredCount)
	}
	b.logger.Infof("Found %d files to link from repository root directory to %s", len(*plan)-start, userHome)
	return nil
}

func (b *linkPlanBuilder) appendDirectoryPlan(plan *linkPlan, repoRoot, srcDir, destDir string, matcher, directoryLinks *ignoreMatcher) error {
	srcPath := filepath.Join(repoRoot, srcDir)
	if !b.fs.DirectoryExists(srcPath) {
		b.logger.Infof("%s directory not found: %s", srcDir, srcPath)
		return nil
	}
	b.logger.Infof("Processing %s directory: %s", srcDir, srcPath)
	start := len(*plan)
	ignoredCount, err := b.collectLinkPlanEntries(plan, repoRoot, srcPath, destDir, matcher, directoryLinks)
	if err != nil {
		return fmt.Errorf("failed to enumerate files in %s: %w", srcDir, err)
	}
	if ignoredCount > 0 {
		b.logger.Infof("Ignored %d entries from %s directory based on ignore patterns", ignoredCount, srcDir)
	}
	b.logger.Infof("Found %d entries to link from %s directory to %s", len(*plan)-start, srcDir, destDir)
	return nil
}

func (b *linkPlanBuilder) collectLinkPlanEntries(plan *linkPlan, repoRoot, sourceRoot, destinationRoot string, matcher, directoryLinks *ignoreMatcher) (int, error) {
	pendingDirectories := []string{sourceRoot}
	ignoredCount := 0
	for len(pendingDirectories) > 0 {
		last := len(pendingDirectories) - 1
		currentDirectory := pendingDirectories[last]
		pendingDirectories = pendingDirectories[:last]
		children, err := b.fs.ReadDirectory(currentDirectory)
		if err != nil {
			return 0, err
		}
		for _, child := range children {
			repositoryRelativePath, err := filepath.Rel(repoRoot, child.Path)
			if err != nil {
				return 0, err
			}
			if child.IsDirectory {
				if shouldIgnoreFile(repositoryRelativePath, true, matcher) {
					ignoredCount++
					b.logger.Verbosef("  Ignored file: %s (matched ignore pattern)", child.Path)
					continue
				}
				if !directoryLinks.ignored(filepath.ToSlash(repositoryRelativePath), true) {
					pendingDirectories = append(pendingDirectories, child.Path)
					continue
				}
				// A selected directory is one operation; never traverse its contents.
			}
			if !child.IsDirectory && shouldIgnoreFile(repositoryRelativePath, false, matcher) {
				ignoredCount++
				b.logger.Verbosef("  Ignored file: %s (matched ignore pattern)", child.Path)
				continue
			}
			relativeTarget, err := filepath.Rel(sourceRoot, child.Path)
			if err != nil {
				return 0, err
			}
			*plan = append(*plan, linkPlanEntry{source: child.Path, target: filepath.Join(destinationRoot, relativeTarget), ensureParent: true})
		}
	}
	return ignoredCount, nil
}

func (b *linkPlanBuilder) loadIgnorePatterns(ignoreFilePath string) ([]string, error) {
	return b.loadPatterns(ignoreFilePath, "ignore")
}

func (b *linkPlanBuilder) loadPatterns(ignoreFilePath, kind string) ([]string, error) {
	exists, err := b.fs.PathExists(ignoreFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect %s file %s: %w", kind, ignoreFilePath, err)
	}
	if !exists {
		b.logger.Verbosef("Optional %s file not found: %s", kind, ignoreFilePath)
		return nil, nil
	}
	lines, err := b.fs.ReadAllLines(ignoreFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s file %s: %w", kind, ignoreFilePath, err)
	}
	return lines, nil
}

func shouldIgnoreFile(filePath string, isDir bool, matcher *ignoreMatcher) bool {
	normalizedPath := filepath.ToSlash(filePath)
	return defaultIgnoreMatcher.ignored(normalizedPath, isDir) || matcher.ignored(normalizedPath, isDir)
}
