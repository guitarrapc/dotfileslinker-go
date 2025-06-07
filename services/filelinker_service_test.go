package services

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/guitarrapc/dotfileslinker-go/infrastructure"
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
	repoRoot := "/repo"
	userHome := "/home/user"
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

		// FileExists or DirectoryExists must return true for the target
		// because linkFile checks target existence before checking if it's a symlink
		fs.AddFile(target, "") // This makes FileExists return true

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

func TestFileLinkerService_LoadIgnoreList(t *testing.T) {
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
		ignoreList := service.loadIgnoreList(ignoreFilePath)

		// Check that expected items are included
		expectedItems := []string{".git", ".ignore", "README.md", "# comment"}
		for _, item := range expectedItems {
			if !ignoreList[item] {
				t.Errorf("Ignore list missing '%s'", item)
			}
		}

		// Check that empty lines are NOT included
		if ignoreList[""] {
			t.Error("Ignore list contains empty line")
		}

		// Count should match expected items (including comment line)
		if len(ignoreList) != len(expectedItems) {
			t.Errorf("Ignore list count mismatch: expected %d, got %d", len(expectedItems), len(ignoreList))
		}
	})

	t.Run("No ignore file", func(t *testing.T) {
		// Create new mocks and service
		fs := infrastructure.NewMockFileSystem()
		logger := NewMockLogger()
		service := NewFileLinkerService(fs, logger)

		// No ignore file setup

		ignoreList := service.loadIgnoreList(ignoreFilePath)

		// Should return empty map
		if len(ignoreList) != 0 {
			t.Errorf("Expected empty ignore list but got: %v", ignoreList)
		}
	})
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
