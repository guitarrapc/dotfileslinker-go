package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/guitarrapc/dotfileslinker-go/infrastructure"
	"github.com/guitarrapc/dotfileslinker-go/services"
)

// Version information set by GoReleaser at build time
var (
	version = "dev"
)

func main() {
	args := os.Args[1:]

	// parse args
	showHelp := containsFlag(args, "--help", "-h")
	showVersion := containsFlag(args, "--version")
	forceOverwrite := containsFlag(args, "--force=y")
	verbose := containsFlag(args, "--verbose", "-v")
	dryRun := containsFlag(args, "--dry-run", "-d")

	// display help or version information and exit if requested
	if showHelp {
		displayHelp()
		return
	}
	if showVersion {
		displayVersion()
		return
	}

	// build up
	fs := infrastructure.NewDefaultFileSystem()
	logger := services.NewConsoleLogger(verbose)
	svc := services.NewFileLinkerService(fs, logger)

	// Get configuration from environment variables or use defaults
	executionRoot := getEnvOrDefault("DOTFILES_ROOT", getCurrentDir())
	userHome := getEnvOrDefault("DOTFILES_HOME", getUserHomeDir())
	ignoreFileName := getEnvOrDefault("DOTFILES_IGNORE_FILE", "dotfiles_ignore")

	logger.Info(fmt.Sprintf("Execution root: %s", executionRoot))
	logger.Info(fmt.Sprintf("User home: %s", userHome))
	logger.Info(fmt.Sprintf("Ignore file: %s", ignoreFileName))
	logger.Info(fmt.Sprintf("Force overwrite: %v", forceOverwrite))
	logger.Info(fmt.Sprintf("Dry run: %v", dryRun))

	// execute
	err := svc.LinkDotfiles(executionRoot, userHome, ignoreFileName, forceOverwrite, dryRun)
	if err != nil {
		handleError(logger, err)
		os.Exit(1)
	}

	if dryRun {
		logger.Success("Dry run completed successfully. No changes were made.")
	} else {
		logger.Success("All operations completed.")
	}
}

// handleError logs errors based on their type
func handleError(logger services.Logger, err error) {
	switch {
	case os.IsPermission(err):
		logger.Error("Permission denied: " + err.Error())
	case os.IsNotExist(err):
		if strings.Contains(err.Error(), "file") {
			logger.Error("File not found: " + err.Error())
		} else {
			logger.Error("Directory not found: " + err.Error())
		}
	default:
		logger.Error("An unexpected error occurred: " + err.Error())
	}
}

// containsFlag checks if args contains any of the specified flags
func containsFlag(args []string, flags ...string) bool {
	for _, arg := range args {
		for _, flag := range flags {
			if strings.EqualFold(arg, flag) {
				return true
			}
		}
	}
	return false
}

// getEnvOrDefault gets an environment variable or returns a default value if not set
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getCurrentDir gets the current working directory
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

// getUserHomeDir gets the user's home directory
func getUserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback for older Go versions
		if runtime.GOOS == "windows" {
			home = os.Getenv("USERPROFILE")
		} else {
			home = os.Getenv("HOME")
		}
	}
	return home
}

// displayHelp displays help information for the application
func displayHelp() {
	appName := filepath.Base(os.Args[0])
	fmt.Printf(`Dotfiles Linker - A utility to link dotfiles from a repository to your home directory

Usage: %s [options]

Options:
  --help, -h         Display this help message
  --force=y          Overwrite existing files or directories
  --verbose, -v      Display detailed information during execution
  --version          Display version information
  --dry-run, -d      Simulate the operations without making any changes

Description:
  This utility creates symbolic links from files in the current directory
  to the appropriate locations in your home directory.

Directory Structure:
  - Files with a '.' prefix in the repository root will be linked directly to $HOME
  - Files in the HOME/ directory will be linked to the same relative path in $HOME
  - Files in the ROOT/ directory will be linked to the same relative path in /
    (Only available on Linux/macOS)

Ignore File:
  Files listed in 'dotfiles_ignore' will be excluded from linking

Environment Variables:
  DOTFILES_ROOT            Directory containing dotfiles (default: current directory)
  DOTFILES_HOME            Target home directory (default: user's home directory)
  DOTFILES_IGNORE_FILE     Name of ignore file (default: dotfiles_ignore)

Examples:
  %s              # Link dotfiles using default settings
  %s --force=y    # Overwrite any existing files
  %s --verbose    # Show detailed information
  %s --dry-run    # Simulate the operations
`, appName, appName, appName, appName, appName)
}

// displayVersion displays version information for the application
func displayVersion() {
	appName := filepath.Base(os.Args[0])
	fmt.Printf("%s version %s\n", appName, version)
}
