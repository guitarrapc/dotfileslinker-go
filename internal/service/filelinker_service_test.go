package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guitarrapc/dotfileslinker-go/internal/infrastructure"
)

// Mock logger for testing
type MockLogger struct {
	SuccessLogs []string
	ErrorLogs   []string
	InfoLogs    []string
	VerboseLogs []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		SuccessLogs: []string{},
		ErrorLogs:   []string{},
		InfoLogs:    []string{},
		VerboseLogs: []string{},
	}
}

func (ml *MockLogger) Success(message string) {
	ml.SuccessLogs = append(ml.SuccessLogs, message)
}

func (ml *MockLogger) Error(message string) {
	ml.ErrorLogs = append(ml.ErrorLogs, message)
}

func (ml *MockLogger) Info(message string) {
	ml.InfoLogs = append(ml.InfoLogs, message)
}

func (ml *MockLogger) Infof(format string, arguments ...any) {
	ml.InfoLogs = append(ml.InfoLogs, fmt.Sprintf(format, arguments...))
}

func (ml *MockLogger) Verbose(message string) {
	ml.VerboseLogs = append(ml.VerboseLogs, message)
}

func (ml *MockLogger) Verbosef(format string, arguments ...any) {
	ml.VerboseLogs = append(ml.VerboseLogs, fmt.Sprintf(format, arguments...))
}

// Tests for FileLinkerService
func TestFileLinkerService_LinkDotfiles(t *testing.T) {
	// Setup test environment
	fs := infrastructure.NewMockFileSystem()
	logger := NewMockLogger()

	// Basic path settings for tests
	testRoot := filepath.Join(os.TempDir(), "dotfileslinker", "link-dotfiles")
	repoRoot := filepath.Join(testRoot, "repo")
	userHome := filepath.Join(testRoot, "home", "user")
	ignoreFileName := ".ignore"

	// Set up files and directory structure for testing
	fs.AddFile(filepath.Join(repoRoot, ".bashrc"), "# bashrc content")
	fs.AddFile(filepath.Join(repoRoot, ".vimrc"), "# vimrc content")
	fs.AddFile(filepath.Join(repoRoot, ignoreFileName), ".git\n.ignore\nREADME.md")
	fs.AddFile(filepath.Join(repoRoot, "README.md"), "# readme")
	fs.AddDirectory(filepath.Join(repoRoot, "HOME"))
	fs.AddFile(filepath.Join(repoRoot, "HOME", ".config", "nvim", "init.vim"), "# neovim config")
	fs.AddDirectory(filepath.Join(repoRoot, "ROOT"))
	fs.AddFile(filepath.Join(repoRoot, "ROOT", "etc", "hosts"), "127.0.0.1 localhost")

	// Configure file enumeration results
	fs.SetupFileEnumeration(repoRoot, ".*", false, []string{
		filepath.Join(repoRoot, ".bashrc"),
		filepath.Join(repoRoot, ".vimrc"),
		filepath.Join(repoRoot, ".ignore"),
		filepath.Join(repoRoot, ".git"),
	})

	fs.SetupFileEnumeration(filepath.Join(repoRoot, "HOME"), "*", true, []string{
		filepath.Join(repoRoot, "HOME", ".config", "nvim", "init.vim"),
	})

	fs.SetupFileEnumeration(filepath.Join(repoRoot, "ROOT"), "*", true, []string{
		filepath.Join(repoRoot, "ROOT", "etc", "hosts"),
	})

	// Create the service for testing
	service := NewFileLinkerService(fs, logger)

	// Run tests
	t.Run("Normal linking operation", func(t *testing.T) {
		err := service.LinkDotfiles(repoRoot, userHome, ignoreFileName, false, false)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check that symbolic links were created
		expectedLinks := map[string]string{
			filepath.Join(userHome, ".bashrc"):                     filepath.Join(repoRoot, ".bashrc"),
			filepath.Join(userHome, ".vimrc"):                      filepath.Join(repoRoot, ".vimrc"),
			filepath.Join(userHome, ".config", "nvim", "init.vim"): filepath.Join(repoRoot, "HOME", ".config", "nvim", "init.vim"),
		}

		for link, target := range expectedLinks {
			if fs.GetLinkTarget(link) != target {
				t.Errorf("Link not correctly created: %s -> %s, actual: %s", link, target, fs.GetLinkTarget(link))
			}
		}

		// Check that ignored files are not linked
		ignoredLink := filepath.Join(userHome, "README.md")
		if fs.GetLinkTarget(ignoredLink) != "" {
			t.Errorf("Ignored file was linked: %s", ignoredLink)
		}

		// Check logs
		if len(logger.SuccessLogs) == 0 {
			t.Error("No success logs output")
		}
	})

	t.Run("Existing files without overwrite", func(t *testing.T) {
		// Create new mocks and service
		fs := infrastructure.NewMockFileSystem()
		logger := NewMockLogger()
		service := NewFileLinkerService(fs, logger)

		// Repository setup
		fs.AddFile(filepath.Join(repoRoot, ".bashrc"), "# repo bashrc")
		fs.SetupFileEnumeration(repoRoot, ".*", false, []string{
			filepath.Join(repoRoot, ".bashrc"),
		})

		// Add existing file
		fs.AddFile(filepath.Join(userHome, ".bashrc"), "# existing bashrc")

		// Execute link operation (without overwrite)
		err := service.LinkDotfiles(repoRoot, userHome, ignoreFileName, false, false)

		// Should get an error with overwrite=false
		if err == nil {
			t.Fatal("Expected error with existing file without overwrite")
		}

		// Ensure no link was created
		if fs.GetLinkTarget(filepath.Join(userHome, ".bashrc")) != "" {
			t.Error("Link created when overwrite=false")
		}
	})

	t.Run("Existing files with overwrite", func(t *testing.T) {
		// Create new mocks and service
		fs := infrastructure.NewMockFileSystem()
		logger := NewMockLogger()
		service := NewFileLinkerService(fs, logger)

		// Repository setup
		fs.AddFile(filepath.Join(repoRoot, ".bashrc"), "# repo bashrc")
		fs.SetupFileEnumeration(repoRoot, ".*", false, []string{
			filepath.Join(repoRoot, ".bashrc"),
		})

		// Add existing file
		fs.AddFile(filepath.Join(userHome, ".bashrc"), "# existing bashrc")

		// Execute link operation (with overwrite)
		err := service.LinkDotfiles(repoRoot, userHome, ignoreFileName, true, false)

		// Should not error with overwrite=true
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Ensure link was created
		target := fs.GetLinkTarget(filepath.Join(userHome, ".bashrc"))
		if target != filepath.Join(repoRoot, ".bashrc") {
			t.Errorf("Link not correctly created: expected %s, got %s", filepath.Join(repoRoot, ".bashrc"), target)
		}
	})

	t.Run("Skip already linked files", func(t *testing.T) {
		// Create new mocks and service
		fs := infrastructure.NewMockFileSystem()
		logger := NewMockLogger()

		// Repository setup
		source := filepath.Join(repoRoot, ".bashrc")
		fs.AddFile(source, "# repo bashrc")
		fs.SetupFileEnumeration(repoRoot, ".*", false, []string{source})

		// Add pre-existing symlink to the same target
		target := filepath.Join(userHome, ".bashrc")
		fs.SymLinks[target] = source

		// Create service with the prepared mocks
		service := NewFileLinkerService(fs, logger)

		// Execute link operation
		err := service.LinkDotfiles(repoRoot, userHome, ignoreFileName, false, false)

		// Should not error
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Ensure link is maintained
		if fs.GetLinkTarget(target) != source {
			t.Errorf("Existing correct link was changed: expected %s, got %s", source, fs.GetLinkTarget(target))
		}

		// Verify skip message - Note: The actual message format is "target -> source"
		hasSkipMsg := false
		for _, msg := range logger.SuccessLogs {
			if strings.Contains(msg, "Skipping already linked") &&
				strings.Contains(msg, target) &&
				strings.Contains(msg, source) {
				hasSkipMsg = true
				break
			}
		}

		if !hasSkipMsg {
			// Print all success logs to help debug
			t.Logf("All success logs: %v", logger.SuccessLogs)
			t.Error("Skip log not found")
		}
	})
}

func TestLinkDotfilesResolvesRelativeRepositoryRootBeforeCreatingLinks(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	relativeRepoRoot := filepath.Join("testdata", "relative-repository")
	absoluteRepoRoot, err := filepath.Abs(relativeRepoRoot)
	if err != nil {
		t.Fatal(err)
	}
	userHome := filepath.Join(os.TempDir(), "dotfileslinker", "relative-root-home")
	source := filepath.Join(absoluteRepoRoot, ".bashrc")
	target := filepath.Join(userHome, ".bashrc")
	fs.AddFile(source, "bashrc")
	fs.SetupFileEnumeration(absoluteRepoRoot, ".*", false, []string{source})
	service := NewFileLinkerService(fs, NewMockLogger())

	if err := service.LinkDotfiles(relativeRepoRoot, userHome, "dotfiles_ignore", false, false); err != nil {
		t.Fatalf("LinkDotfiles() error = %v", err)
	}
	if got := fs.GetLinkTarget(target); got != source {
		t.Fatalf("link target = %q, want absolute source %q", got, source)
	}
	if !filepath.IsAbs(fs.GetLinkTarget(target)) {
		t.Fatalf("link target is relative: %q", fs.GetLinkTarget(target))
	}
}

func TestLinkDotfilesRejectsUserHomeInsideRepositoryBeforeProcessing(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	repoRoot := filepath.Join(os.TempDir(), "dotfileslinker", "overlap", "repo")
	userHome := filepath.Join(repoRoot, "home")
	service := NewFileLinkerService(fs, NewMockLogger())

	err := service.LinkDotfiles(repoRoot, userHome, "dotfiles_ignore", true, false)
	if err == nil || !strings.Contains(err.Error(), "must not be the repository root") {
		t.Fatalf("LinkDotfiles() error = %v, want repository overlap error", err)
	}
	if len(fs.OperationLog) != 0 {
		t.Fatalf("filesystem was accessed before root validation: %v", fs.OperationLog)
	}
}

func TestLinkDotfilesSameRootWithForcePreservesRealSourceFile(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, ".settings")
	if err := os.WriteFile(source, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewFileLinkerService(infrastructure.NewDefaultFileSystem(), NewNullLogger())

	err := service.LinkDotfiles(root, root, "dotfiles_ignore", true, false)
	if err == nil || !strings.Contains(err.Error(), "must not be the repository root") {
		t.Fatalf("LinkDotfiles() error = %v, want repository overlap error", err)
	}
	content, readErr := os.ReadFile(source)
	if readErr != nil {
		t.Fatalf("source file was removed: %v", readErr)
	}
	if string(content) != "original" {
		t.Fatalf("source content = %q, want original", content)
	}
	if target, readLinkErr := os.Readlink(source); readLinkErr == nil {
		t.Fatalf("source was replaced with symlink to %q", target)
	}
}

func TestLinkDotfilesValidatesEntirePlanBeforeMutation(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	root := filepath.Join(os.TempDir(), "dotfileslinker", "plan-overlap")
	userHome := root
	repoRoot := filepath.Join(root, ".dotfiles", "repository")
	validSource := filepath.Join(repoRoot, ".valid")
	overlappingSource := filepath.Join(repoRoot, ".dotfiles")
	fs.AddFile(validSource, "valid")
	fs.AddFile(overlappingSource, "overlap")
	fs.SetupFileEnumeration(repoRoot, ".*", false, []string{validSource, overlappingSource})
	service := NewFileLinkerService(fs, NewMockLogger())

	err := service.LinkDotfiles(repoRoot, userHome, "dotfiles_ignore", true, false)
	if err == nil || !strings.Contains(err.Error(), "overlaps dotfiles repository") {
		t.Fatalf("LinkDotfiles() error = %v, want destination overlap error", err)
	}
	assertNoMutationOperations(t, fs.OperationLog)
}

func TestLinkDotfilesRejectsIdenticalSourceAndDestinationBeforeMutation(t *testing.T) {
	root := filepath.Join(os.TempDir(), "dotfileslinker", "same-path")
	repoRoot := filepath.Join(root, "repo")
	userHome := filepath.Join(root, "home")
	sourceAndTarget := filepath.Join(userHome, ".settings")

	err := validateLinkPlan(repoRoot, []linkPlanEntry{{source: sourceAndTarget, target: sourceAndTarget}})
	if err == nil || !strings.Contains(err.Error(), "same path") {
		t.Fatalf("validateLinkPlan() error = %v, want same path error", err)
	}
}

func TestLinkDotfilesRejectsDuplicateDestinationBeforeMutation(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	root := filepath.Join(os.TempDir(), "dotfileslinker", "duplicate-target")
	repoRoot := filepath.Join(root, "repo")
	userHome := filepath.Join(root, "home")
	rootSource := filepath.Join(repoRoot, ".config")
	homeRoot := filepath.Join(repoRoot, "HOME")
	homeSource := filepath.Join(homeRoot, ".config")
	fs.AddFile(rootSource, "root")
	fs.AddDirectory(homeRoot)
	fs.AddFile(homeSource, "home")
	fs.SetupFileEnumeration(repoRoot, ".*", false, []string{rootSource})
	service := NewFileLinkerService(fs, NewMockLogger())

	err := service.LinkDotfiles(repoRoot, userHome, "dotfiles_ignore", true, false)
	if err == nil || !strings.Contains(err.Error(), "multiple sources map") {
		t.Fatalf("LinkDotfiles() error = %v, want duplicate destination error", err)
	}
	assertNoMutationOperations(t, fs.OperationLog)
}

func TestLinkDotfilesValidatesAllExistingTargetsBeforeApplyingPlan(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	root := filepath.Join(os.TempDir(), "dotfileslinker", "preflight-conflict")
	repoRoot := filepath.Join(root, "repo")
	userHome := filepath.Join(root, "home")
	firstSource := filepath.Join(repoRoot, ".first")
	secondSource := filepath.Join(repoRoot, ".second")
	secondTarget := filepath.Join(userHome, ".second")
	fs.AddFile(firstSource, "first")
	fs.AddFile(secondSource, "second")
	fs.AddFile(secondTarget, "existing")
	fs.SetupFileEnumeration(repoRoot, ".*", false, []string{firstSource, secondSource})
	service := NewFileLinkerService(fs, NewMockLogger())

	err := service.LinkDotfiles(repoRoot, userHome, "dotfiles_ignore", false, false)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("LinkDotfiles() error = %v, want existing-target conflict", err)
	}
	assertNoMutationOperations(t, fs.OperationLog)
}

func TestLinkDotfilesDryRunReportsRemainingEntriesAfterConflict(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	logger := NewMockLogger()
	root := filepath.Join(os.TempDir(), "dotfileslinker", "dry-run-conflict")
	repoRoot := filepath.Join(root, "repo")
	userHome := filepath.Join(root, "home")
	conflictingSource := filepath.Join(repoRoot, ".conflict")
	remainingSource := filepath.Join(repoRoot, ".remaining")
	conflictingTarget := filepath.Join(userHome, ".conflict")
	remainingTarget := filepath.Join(userHome, ".remaining")
	fs.AddFile(conflictingSource, "conflict")
	fs.AddFile(remainingSource, "remaining")
	fs.AddFile(conflictingTarget, "existing")
	fs.SetupFileEnumeration(repoRoot, ".*", false, []string{conflictingSource, remainingSource})
	service := NewFileLinkerService(fs, logger)

	err := service.LinkDotfiles(repoRoot, userHome, "dotfiles_ignore", false, true)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("LinkDotfiles() error = %v, want existing-target conflict", err)
	}
	if !containsLog(logger.ErrorLogs, conflictingTarget) {
		t.Fatalf("dry-run did not report conflict for %s: %v", conflictingTarget, logger.ErrorLogs)
	}
	if !containsLog(logger.SuccessLogs, remainingTarget) {
		t.Fatalf("dry-run stopped before reporting %s: %v", remainingTarget, logger.SuccessLogs)
	}
	assertNoMutationOperations(t, fs.OperationLog)
}

func TestLinkDotfilesRollsBackEarlierLinksWhenLaterCreationFails(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	root := filepath.Join(os.TempDir(), "dotfileslinker", "apply-rollback")
	repoRoot := filepath.Join(root, "repo")
	userHome := filepath.Join(root, "home")
	firstSource := filepath.Join(repoRoot, ".first")
	secondSource := filepath.Join(repoRoot, ".second")
	firstTarget := filepath.Join(userHome, ".first")
	secondTarget := filepath.Join(userHome, ".second")
	creationError := errors.New("creation failed")
	fs.AddFile(firstSource, "first")
	fs.AddFile(secondSource, "second")
	fs.SetupFileEnumeration(repoRoot, ".*", false, []string{firstSource, secondSource})
	fs.SetErrorForOperation("CreateFileSymlink:"+secondTarget, creationError)
	service := NewFileLinkerService(fs, NewMockLogger())

	err := service.LinkDotfiles(repoRoot, userHome, "dotfiles_ignore", false, false)
	if !errors.Is(err, creationError) {
		t.Fatalf("LinkDotfiles() error = %v, want wrapped %v", err, creationError)
	}
	if _, exists := fs.SymLinks[firstTarget]; exists {
		t.Fatalf("earlier link remains after rollback: %s", firstTarget)
	}
	if !containsOperation(fs.OperationLog, "Delete: "+firstTarget) {
		t.Fatalf("earlier link was not rolled back: %v", fs.OperationLog)
	}
}

func TestLinkDotfilesRestoresEarlierReplacementWhenLaterCreationFails(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	root := filepath.Join(os.TempDir(), "dotfileslinker", "replacement-rollback")
	repoRoot := filepath.Join(root, "repo")
	userHome := filepath.Join(root, "home")
	firstSource := filepath.Join(repoRoot, ".first")
	secondSource := filepath.Join(repoRoot, ".second")
	firstTarget := filepath.Join(userHome, ".first")
	secondTarget := filepath.Join(userHome, ".second")
	firstBackup := firstTarget + ".dotfileslinker-backup"
	creationError := errors.New("creation failed")
	fs.AddFile(firstSource, "new first")
	fs.AddFile(secondSource, "second")
	fs.AddFile(firstTarget, "original first")
	fs.SetupFileEnumeration(repoRoot, ".*", false, []string{firstSource, secondSource})
	fs.SetErrorForOperation("CreateFileSymlink:"+secondTarget, creationError)
	service := NewFileLinkerService(fs, NewMockLogger())

	err := service.LinkDotfiles(repoRoot, userHome, "dotfiles_ignore", true, false)
	if !errors.Is(err, creationError) {
		t.Fatalf("LinkDotfiles() error = %v, want wrapped %v", err, creationError)
	}
	if got := fs.Files[firstTarget]; got != "original first" {
		t.Fatalf("earlier replacement was not restored: got %q", got)
	}
	if _, exists := fs.SymLinks[firstTarget]; exists {
		t.Fatalf("replacement link remains after rollback: %s", firstTarget)
	}
	if _, exists := fs.Files[firstBackup]; exists {
		t.Fatalf("backup remains after successful rollback: %s", firstBackup)
	}
}

func TestExecuteLinkPlanPreparesAllParentsBeforeCreatingLinks(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	homeSource := filepath.Clean("/repo/HOME/.config/app/config")
	rootSource := filepath.Clean("/repo/ROOT/etc/app/config")
	homeTarget := filepath.Clean("/home/user/.config/app/config")
	rootTarget := filepath.Clean("/etc/app/config")
	rootParent := filepath.Dir(rootTarget)
	permissionError := errors.New("permission denied")
	fs.AddFile(homeSource, "home")
	fs.AddFile(rootSource, "root")
	fs.SetErrorForOperation("EnsureDirectory:"+rootParent, permissionError)
	service := NewFileLinkerService(fs, NewMockLogger())
	plan := []validatedLinkPlanEntry{
		{linkPlanEntry: linkPlanEntry{source: homeSource, target: homeTarget, ensureParent: true}},
		{linkPlanEntry: linkPlanEntry{source: rootSource, target: rootTarget, ensureParent: true}},
	}

	err := service.executeLinkPlan(plan, false)
	if !errors.Is(err, permissionError) {
		t.Fatalf("executeLinkPlan() error = %v, want wrapped %v", err, permissionError)
	}
	for _, operation := range fs.OperationLog {
		if strings.HasPrefix(operation, "CreateFileSymlink:") || strings.HasPrefix(operation, "CreateDirectorySymlink:") {
			t.Fatalf("link was created before every parent was prepared: %v", fs.OperationLog)
		}
	}
	if _, exists := fs.Directories[filepath.Dir(homeTarget)]; exists {
		t.Fatalf("prepared parent remains after later preparation failed: %v", fs.Directories)
	}
}

func TestExecuteLinkPlanRollsBackCreatedParentsWhenLinkFails(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	source := filepath.Clean("/repo/HOME/.config/app/config")
	userHome := filepath.Clean("/home/user")
	target := filepath.Join(userHome, ".config", "app", "config")
	createdParent := filepath.Dir(target)
	createdAncestor := filepath.Dir(createdParent)
	creationError := errors.New("link creation failed")
	fs.AddFile(source, "content")
	fs.AddDirectory(userHome)
	fs.SetErrorForOperation("CreateFileSymlink:"+target, creationError)
	service := NewFileLinkerService(fs, NewMockLogger())
	plan := []validatedLinkPlanEntry{{
		linkPlanEntry: linkPlanEntry{source: source, target: target, ensureParent: true},
	}}

	err := service.executeLinkPlan(plan, false)
	if !errors.Is(err, creationError) {
		t.Fatalf("executeLinkPlan() error = %v, want wrapped %v", err, creationError)
	}
	for _, directory := range []string{createdParent, createdAncestor} {
		if _, exists := fs.Directories[directory]; exists {
			t.Errorf("created directory remains after rollback: %s", directory)
		}
	}
	if _, exists := fs.Directories[userHome]; !exists {
		t.Fatalf("pre-existing directory was removed during rollback: %s", userHome)
	}

	deleteParent := operationIndex(fs.OperationLog, "Delete: "+createdParent)
	deleteAncestor := operationIndex(fs.OperationLog, "Delete: "+createdAncestor)
	if deleteParent < 0 || deleteAncestor < 0 || deleteParent > deleteAncestor {
		t.Fatalf("created directories were not removed deepest first: %v", fs.OperationLog)
	}
}

func containsLog(logs []string, substring string) bool {
	for _, log := range logs {
		if strings.Contains(log, substring) {
			return true
		}
	}
	return false
}

func operationIndex(operations []string, want string) int {
	for index, operation := range operations {
		if operation == want {
			return index
		}
	}
	return -1
}

func assertNoMutationOperations(t *testing.T, operations []string) {
	t.Helper()
	for _, operation := range operations {
		for _, prefix := range []string{"Move:", "Delete:", "EnsureDirectory:", "CreateFileSymlink:", "CreateDirectorySymlink:"} {
			if strings.HasPrefix(operation, prefix) {
				t.Fatalf("mutation occurred before plan validation: %v", operations)
			}
		}
	}
}

func TestLinkFileSkipsEquivalentRelativeSymlink(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	logger := NewMockLogger()
	root := filepath.Join(os.TempDir(), "dotfileslinker", "relative-link-service")
	source := filepath.Join(root, "repo", ".settings")
	target := filepath.Join(root, "home", ".settings")
	relativeTarget, err := filepath.Rel(filepath.Dir(target), source)
	if err != nil {
		t.Fatal(err)
	}
	fs.AddFile(source, "settings")
	fs.SymLinks[target] = relativeTarget
	service := NewFileLinkerService(fs, logger)

	if err := service.linkFile(source, target, false, false); err != nil {
		t.Fatalf("linkFile() error = %v", err)
	}
	if got := fs.GetLinkTarget(target); got != relativeTarget {
		t.Errorf("relative link target changed: got %q, want %q", got, relativeTarget)
	}
	for _, operation := range fs.OperationLog {
		if operation == "Delete: "+target {
			t.Fatalf("equivalent relative symlink was deleted: %s", operation)
		}
	}
}

func TestLinkFileDanglingSymlink(t *testing.T) {
	tests := []struct {
		name      string
		overwrite bool
		wantError bool
		wantLink  string
	}{
		{name: "requires force", overwrite: false, wantError: true, wantLink: "/missing"},
		{name: "repaired with force", overwrite: true, wantError: false, wantLink: "/repo/.bashrc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := infrastructure.NewMockFileSystem()
			source := filepath.Clean("/repo/.bashrc")
			target := filepath.Clean("/home/user/.bashrc")
			fs.AddFile(source, "bashrc")
			fs.SymLinks[target] = filepath.Clean("/missing")
			service := NewFileLinkerService(fs, NewMockLogger())

			err := service.linkFile(source, target, tt.overwrite, false)
			if (err != nil) != tt.wantError {
				t.Fatalf("linkFile() error = %v, wantError %v", err, tt.wantError)
			}
			if got := fs.GetLinkTarget(target); got != filepath.Clean(tt.wantLink) {
				t.Errorf("link target = %q, want %q", got, filepath.Clean(tt.wantLink))
			}
		})
	}
}

func TestLinkFileReturnsPathInspectionError(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	source := filepath.Clean("/repo/.bashrc")
	target := filepath.Clean("/home/user/.bashrc")
	inspectionError := errors.New("inspection failed")
	fs.AddFile(source, "bashrc")
	fs.SetErrorForOperation("PathExists:"+target, inspectionError)
	service := NewFileLinkerService(fs, NewMockLogger())

	err := service.linkFile(source, target, true, false)
	if !errors.Is(err, inspectionError) {
		t.Fatalf("linkFile() error = %v, want wrapped %v", err, inspectionError)
	}
	if fs.GetLinkTarget(target) != "" {
		t.Fatal("link was created after target inspection failed")
	}
}

func TestLinkFileRestoresExistingTargetWhenLinkCreationFails(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	source := filepath.Clean("/repo/.bashrc")
	target := filepath.Clean("/home/user/.bashrc")
	backup := target + ".dotfileslinker-backup"
	creationError := errors.New("symlink creation failed")
	fs.AddFile(source, "new")
	fs.AddFile(target, "original")
	fs.SetErrorForOperation("CreateFileSymlink:"+target, creationError)
	service := NewFileLinkerService(fs, NewMockLogger())

	err := service.linkFile(source, target, true, false)
	if !errors.Is(err, creationError) {
		t.Fatalf("linkFile() error = %v, want wrapped %v", err, creationError)
	}
	if got := fs.Files[target]; got != "original" {
		t.Fatalf("original target was not restored: got %q", got)
	}
	if _, exists := fs.Files[backup]; exists {
		t.Fatal("temporary backup remains after successful rollback")
	}
	if got := fs.GetLinkTarget(target); got != "" {
		t.Fatalf("failed replacement remains at target: %q", got)
	}

	wantOperations := []string{
		"Move: " + target + " -> " + backup,
		"Move: " + backup + " -> " + target,
	}
	for _, want := range wantOperations {
		if !containsOperation(fs.OperationLog, want) {
			t.Errorf("operation log does not contain %q: %v", want, fs.OperationLog)
		}
	}
}

func TestLinkFileRemovesBackupAfterSuccessfulReplacement(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	source := filepath.Clean("/repo/.bashrc")
	target := filepath.Clean("/home/user/.bashrc")
	backup := target + ".dotfileslinker-backup"
	fs.AddFile(source, "new")
	fs.AddFile(target, "original")
	service := NewFileLinkerService(fs, NewMockLogger())

	if err := service.linkFile(source, target, true, false); err != nil {
		t.Fatalf("linkFile() error = %v", err)
	}
	if got := fs.GetLinkTarget(target); got != source {
		t.Fatalf("link target = %q, want %q", got, source)
	}
	if _, exists := fs.Files[backup]; exists {
		t.Fatal("temporary backup remains after successful replacement")
	}
	if !containsOperation(fs.OperationLog, "RemoveAll: "+backup) {
		t.Fatalf("backup was not deleted: %v", fs.OperationLog)
	}
}

func TestLinkFileReplacesNonEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := infrastructure.NewDefaultFileSystem()
	probe := filepath.Join(root, "symlink-probe")
	if err := fs.CreateFileSymlink(probe, source); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if err := fs.Delete(probe); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(root, "target")
	nestedFile := filepath.Join(target, "nested", "original.txt")
	if err := os.MkdirAll(filepath.Dir(nestedFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nestedFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewFileLinkerService(fs, NewNullLogger())
	if err := service.linkFile(source, target, true, false); err != nil {
		t.Fatalf("linkFile() error = %v", err)
	}
	if got := fs.GetLinkTarget(target); got != source {
		t.Fatalf("link target = %q, want %q", got, source)
	}
	backup := target + ".dotfileslinker-backup"
	if exists, err := fs.PathExists(backup); err != nil || exists {
		t.Fatalf("replacement backup remains: exists = %v, error = %v", exists, err)
	}
}

func TestLinkFileLeavesAppliedLinkAndBackupWhenBackupCleanupFails(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	source := filepath.Clean("/repo/.bashrc")
	target := filepath.Clean("/home/user/.bashrc")
	backup := target + ".dotfileslinker-backup"
	cleanupError := errors.New("cleanup failed")
	fs.AddFile(source, "new")
	fs.AddFile(target, "original")
	fs.SetErrorForOperation("RemoveAll:"+backup, cleanupError)
	service := NewFileLinkerService(fs, NewMockLogger())

	err := service.linkFile(source, target, true, false)
	if !errors.Is(err, cleanupError) {
		t.Fatalf("linkFile() error = %v, want wrapped %v", err, cleanupError)
	}
	if got := fs.GetLinkTarget(target); got != source {
		t.Fatalf("applied link target = %q, want %q", got, source)
	}
	if got := fs.Files[backup]; got != "original" {
		t.Fatalf("preserved backup = %q, want original content", got)
	}
}

func containsOperation(operations []string, want string) bool {
	for _, operation := range operations {
		if operation == want {
			return true
		}
	}
	return false
}

func TestFileLinkerService_LoadIgnorePatterns(t *testing.T) {
	// Setup test environment
	fs := infrastructure.NewMockFileSystem()
	logger := NewMockLogger()

	// Basic path settings for testing
	repoRoot := "/repo"
	ignoreFileName := ".ignore"
	ignoreFilePath := filepath.Join(repoRoot, ignoreFileName)

	// Setup ignore file with empty lines and comment lines
	fs.AddFile(ignoreFilePath, ".git\n.ignore\nREADME.md\n\n# comment\n")

	// Create the service for testing
	service := NewFileLinkerService(fs, logger)

	t.Run("Loading ignore list", func(t *testing.T) {
		patterns, err := service.loadIgnorePatterns(ignoreFilePath)
		if err != nil {
			t.Fatalf("loadIgnorePatterns() error = %v", err)
		}
		expected := []string{".git", ".ignore", "README.md", "", "# comment", ""}

		if len(patterns) != len(expected) {
			t.Fatalf("pattern count mismatch: got %d, want %d", len(patterns), len(expected))
		}
		for i := range expected {
			if patterns[i] != expected[i] {
				t.Errorf("pattern %d = %q, want %q", i, patterns[i], expected[i])
			}
		}
	})

	t.Run("No ignore file", func(t *testing.T) {
		// Create new mocks and service
		fs := infrastructure.NewMockFileSystem()
		logger := NewMockLogger()
		service := NewFileLinkerService(fs, logger)

		// No ignore file setup

		patterns, err := service.loadIgnorePatterns(ignoreFilePath)
		if err != nil {
			t.Fatalf("loadIgnorePatterns() error = %v", err)
		}

		if len(patterns) != 0 {
			t.Errorf("expected no ignore patterns, got: %v", patterns)
		}
	})
}

func TestLinkDotfilesStopsWhenIgnoreFileCannotBeRead(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	testRoot := filepath.Join(os.TempDir(), "dotfileslinker", "ignore-read-error")
	repoRoot := filepath.Join(testRoot, "repo")
	userHome := filepath.Join(testRoot, "home", "user")
	ignoreFileName := "dotfiles_ignore"
	ignoreFilePath := filepath.Join(repoRoot, ignoreFileName)
	readError := errors.New("access denied")
	fs.AddFile(ignoreFilePath, "*.secret")
	fs.AddFile(filepath.Join(repoRoot, ".secret"), "sensitive")
	fs.SetErrorForOperation("ReadAllLines:"+ignoreFilePath, readError)
	service := NewFileLinkerService(fs, NewMockLogger())

	err := service.LinkDotfiles(repoRoot, userHome, ignoreFileName, false, false)
	if !errors.Is(err, readError) {
		t.Fatalf("LinkDotfiles() error = %v, want wrapped %v", err, readError)
	}
	if !strings.Contains(err.Error(), ignoreFilePath) {
		t.Fatalf("LinkDotfiles() error = %q, want ignore path", err)
	}
	for _, operation := range fs.OperationLog {
		if strings.HasPrefix(operation, "ReadDirectory:") || strings.HasPrefix(operation, "EnumerateFiles:") || strings.HasPrefix(operation, "CreateFileSymlink:") {
			t.Fatalf("processing continued after ignore read error: %v", fs.OperationLog)
		}
	}
}

func TestFileLinkerService_GitIgnoreSemantics(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	service := NewFileLinkerService(fs, NewMockLogger())
	testRoot := filepath.Join(os.TempDir(), "dotfileslinker", "gitignore-semantics")
	repoRoot := filepath.Join(testRoot, "repo")
	userHome := filepath.Join(testRoot, "home", "user")
	ignoreFileName := "dotfiles_ignore"
	homeRoot := filepath.Join(repoRoot, "HOME")

	ignoredByDirectory := filepath.Join(homeRoot, "cache", "data.json")
	ignoredByWildcard := filepath.Join(homeRoot, "logs", "app.log")
	reincludedByLaterRule := filepath.Join(homeRoot, "logs", "important.log")

	fs.AddFile(filepath.Join(repoRoot, ignoreFileName), "# comment\nHOME/cache/\n*.log\n!important.log\n")
	fs.AddDirectory(homeRoot)
	fs.AddFile(ignoredByDirectory, "cache")
	fs.AddFile(ignoredByWildcard, "log")
	fs.AddFile(reincludedByLaterRule, "important")
	fs.SetupFileEnumeration(repoRoot, ".*", false, nil)
	fs.SetupFileEnumeration(homeRoot, "*", true, []string{
		ignoredByDirectory,
		ignoredByWildcard,
		reincludedByLaterRule,
	})

	if err := service.LinkDotfiles(repoRoot, userHome, ignoreFileName, false, false); err != nil {
		t.Fatalf("LinkDotfiles() error = %v", err)
	}

	for _, ignored := range []string{
		filepath.Join(userHome, "cache", "data.json"),
		filepath.Join(userHome, "logs", "app.log"),
	} {
		if target := fs.GetLinkTarget(ignored); target != "" {
			t.Errorf("ignored path %q was linked to %q", ignored, target)
		}
	}

	wantLink := filepath.Join(userHome, "logs", "important.log")
	if target := fs.GetLinkTarget(wantLink); target != reincludedByLaterRule {
		t.Errorf("re-included path target = %q, want %q", target, reincludedByLaterRule)
	}

	forbiddenOperations := map[string]bool{
		"ReadDirectory:" + filepath.Join(homeRoot, "cache"): true,
	}
	for _, operation := range fs.OperationLog {
		if forbiddenOperations[operation] {
			t.Fatalf("ignored directory was traversed: %s", operation)
		}
	}
}

func TestCollectLinkPlanEntriesReadsEachDirectoryOnce(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	repoRoot := filepath.Clean("/repo")
	homeRoot := filepath.Join(repoRoot, "HOME")
	nestedRoot := filepath.Join(homeRoot, ".config")
	fs.AddFile(filepath.Join(nestedRoot, "app", "config.json"), "content")
	service := NewFileLinkerService(fs, NewNullLogger())

	if _, _, err := service.collectLinkPlanEntries(repoRoot, homeRoot, filepath.Clean("/home/user"), newIgnoreMatcher(nil)); err != nil {
		t.Fatalf("collectLinkPlanEntries() error = %v", err)
	}

	readCounts := make(map[string]int)
	for _, operation := range fs.OperationLog {
		if strings.HasPrefix(operation, "ReadDirectory:") {
			readCounts[strings.TrimPrefix(operation, "ReadDirectory:")]++
		}
		if strings.HasPrefix(operation, "EnumerateFiles:") || strings.HasPrefix(operation, "EnumerateDirectories:") {
			t.Fatalf("legacy enumeration was used: %s", operation)
		}
	}
	for directory, count := range readCounts {
		if count != 1 {
			t.Errorf("directory %s was read %d times, want once", directory, count)
		}
	}
	for _, directory := range []string{homeRoot, nestedRoot, filepath.Join(nestedRoot, "app")} {
		if readCounts[directory] != 1 {
			t.Errorf("directory %s read count = %d, want 1", directory, readCounts[directory])
		}
	}
}

// Test dry run functionality
func TestFileLinkerService_DryRun(t *testing.T) {
	// Setup test environment
	fs := infrastructure.NewMockFileSystem()
	logger := NewMockLogger()

	// Basic path settings for tests
	testRoot := filepath.Join(os.TempDir(), "dotfileslinker", "dry-run")
	repoRoot := filepath.Join(testRoot, "repo")
	userHome := filepath.Join(testRoot, "home", "user")
	ignoreFileName := ".ignore"

	// Set up files and directory structure for testing
	fs.AddFile(filepath.Join(repoRoot, ".bashrc"), "# bashrc content")
	fs.AddFile(filepath.Join(repoRoot, ".vimrc"), "# vimrc content")
	fs.AddFile(filepath.Join(repoRoot, ignoreFileName), ".git\n.ignore\nREADME.md\n.DS_Store")
	fs.AddFile(filepath.Join(repoRoot, "README.md"), "# readme")
	fs.AddFile(filepath.Join(repoRoot, ".DS_Store"), "binary content")
	fs.AddDirectory(filepath.Join(repoRoot, "HOME"))
	fs.AddFile(filepath.Join(repoRoot, "HOME", ".config", "nvim", "init.vim"), "# neovim config")
	fs.AddDirectory(filepath.Join(repoRoot, "ROOT"))
	fs.AddFile(filepath.Join(repoRoot, "ROOT", "etc", "hosts"), "127.0.0.1 localhost")

	// Configure file enumeration results
	fs.SetupFileEnumeration(repoRoot, ".*", false, []string{
		filepath.Join(repoRoot, ".bashrc"),
		filepath.Join(repoRoot, ".vimrc"),
		filepath.Join(repoRoot, ".ignore"),
		filepath.Join(repoRoot, ".DS_Store"),
		filepath.Join(repoRoot, ".git"),
	})

	fs.SetupFileEnumeration(filepath.Join(repoRoot, "HOME"), "*", true, []string{
		filepath.Join(repoRoot, "HOME", ".config", "nvim", "init.vim"),
	})

	fs.SetupFileEnumeration(filepath.Join(repoRoot, "ROOT"), "*", true, []string{
		filepath.Join(repoRoot, "ROOT", "etc", "hosts"),
	})

	// Create the service for testing
	service := NewFileLinkerService(fs, logger)

	// Test with dry run
	t.Run("Dry run operation", func(t *testing.T) {
		// Reset the logger
		logger = NewMockLogger()
		service = NewFileLinkerService(fs, logger)

		// Run with dry run enabled
		err := service.LinkDotfiles(repoRoot, userHome, ignoreFileName, false, true)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify dry run logs were produced
		dryRunMsgFound := false
		for _, msg := range logger.InfoLogs {
			if msg == "DRY RUN MODE: No files will be actually linked" {
				dryRunMsgFound = true
				break
			}
		}
		if !dryRunMsgFound {
			t.Error("Dry run mode message not logged")
		}

		// Verify that symbolic links were NOT created
		linksToCheck := []string{
			filepath.Join(userHome, ".bashrc"),
			filepath.Join(userHome, ".vimrc"),
			filepath.Join(userHome, ".config", "nvim", "init.vim"),
		}

		for _, link := range linksToCheck {
			if fs.GetLinkTarget(link) != "" {
				t.Errorf("Link should not be created in dry run mode: %s", link)
			}
		}

		// Verify that [DRY-RUN] prefixed logs were produced
		dryRunOperationFound := false
		for _, msg := range logger.SuccessLogs {
			if strings.HasPrefix(msg, "[DRY-RUN]") {
				dryRunOperationFound = true
				break
			}
		}
		if !dryRunOperationFound {
			t.Error("No dry run operation messages logged")
		}
	})

	// Test OS-specific default ignores
	t.Run("Default OS-specific ignore patterns", func(t *testing.T) {
		// Reset the logger
		logger = NewMockLogger()
		service = NewFileLinkerService(fs, logger)

		// Run with dry run enabled
		err := service.LinkDotfiles(repoRoot, userHome, ignoreFileName, false, true)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify that .DS_Store was ignored
		for _, msg := range logger.VerboseLogs {
			if strings.Contains(msg, ".DS_Store") && strings.Contains(msg, "Ignored file") {
				return // Test passed
			}
		}
		t.Error("OS-specific file (.DS_Store) was not ignored")
	})
}
