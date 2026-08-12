package service

import (
	"errors"
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

func (ml *MockLogger) Verbose(message string) {
	ml.VerboseLogs = append(ml.VerboseLogs, message)
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
	if !containsOperation(fs.OperationLog, "Delete: "+backup) {
		t.Fatalf("backup was not deleted: %v", fs.OperationLog)
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
	repoRoot := filepath.Clean("/repo")
	userHome := filepath.Clean("/home/user")
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
		if strings.HasPrefix(operation, "EnumerateFiles:") || strings.HasPrefix(operation, "CreateFileSymlink:") {
			t.Fatalf("processing continued after ignore read error: %v", fs.OperationLog)
		}
	}
}

func TestFileLinkerService_GitIgnoreSemantics(t *testing.T) {
	fs := infrastructure.NewMockFileSystem()
	service := NewFileLinkerService(fs, NewMockLogger())
	repoRoot := filepath.Clean("/repo")
	userHome := filepath.Clean("/home/user")
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
		"EnumerateDirectories:" + filepath.Join(homeRoot, "cache"):        true,
		"EnumerateFiles:" + filepath.Join(homeRoot, "cache") + ":*:false": true,
	}
	for _, operation := range fs.OperationLog {
		if forbiddenOperations[operation] {
			t.Fatalf("ignored directory was traversed: %s", operation)
		}
	}
}

// Test dry run functionality
func TestFileLinkerService_DryRun(t *testing.T) {
	// Setup test environment
	fs := infrastructure.NewMockFileSystem()
	logger := NewMockLogger()

	// Basic path settings for tests
	repoRoot := "/repo"
	userHome := "/home/user"
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
