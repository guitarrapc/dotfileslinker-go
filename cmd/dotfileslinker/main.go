package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/guitarrapc/dotfileslinker-go/internal/infrastructure"
	"github.com/guitarrapc/dotfileslinker-go/internal/service"
)

// Version information set by GoReleaser at build time
var (
	version = "dev"
)

func main() {
	options, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nTry '%s --help' for more information.\n", err, filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	// display help or version information and exit if requested
	if options.showHelp {
		displayHelp()
		return
	}
	if options.showVersion {
		displayVersion()
		return
	}

	// build up
	fs := infrastructure.NewDefaultFileSystem()
	logger := service.NewConsoleLogger(options.verbose)
	svc := service.NewFileLinkerService(fs, logger)

	// Get configuration from environment variables or use defaults
	executionRoot := getEnvOrDefault("DOTFILES_ROOT", getCurrentDir())
	userHome := getEnvOrDefault("DOTFILES_HOME", getUserHomeDir())
	ignoreFileName := getEnvOrDefault("DOTFILES_IGNORE_FILE", "dotfiles_ignore")

	logger.Info(fmt.Sprintf("Execution root: %s", executionRoot))
	logger.Info(fmt.Sprintf("User home: %s", userHome))
	logger.Info(fmt.Sprintf("Ignore file: %s", ignoreFileName))
	logger.Info(fmt.Sprintf("Force overwrite: %v", options.forceOverwrite))
	logger.Info(fmt.Sprintf("Dry run: %v", options.dryRun))

	// execute
	err = svc.LinkDotfiles(executionRoot, userHome, ignoreFileName, options.forceOverwrite, options.dryRun)
	if err != nil {
		handleError(logger, err)
		os.Exit(1)
	}

	if !options.dryRun {
		logger.Success("All operations completed.")
	}
}

type cliOptions struct {
	showHelp       bool
	showVersion    bool
	forceOverwrite bool
	verbose        bool
	dryRun         bool
}

func parseOptions(args []string) (cliOptions, error) {
	var options cliOptions
	flags := flag.NewFlagSet("dotfileslinker", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.BoolVar(&options.showHelp, "help", false, "display help")
	flags.BoolVar(&options.showHelp, "h", false, "display help")
	flags.BoolVar(&options.showVersion, "version", false, "display version")
	flags.BoolVar(&options.forceOverwrite, "force", false, "overwrite existing files or directories")
	flags.BoolVar(&options.verbose, "verbose", false, "display detailed information")
	flags.BoolVar(&options.verbose, "v", false, "display detailed information")
	flags.BoolVar(&options.dryRun, "dry-run", false, "simulate operations")
	flags.BoolVar(&options.dryRun, "d", false, "simulate operations")

	if err := flags.Parse(args); err != nil {
		return cliOptions{}, err
	}
	if flags.NArg() != 0 {
		return cliOptions{}, fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	return options, nil
}

// handleError logs errors based on their type
func handleError(logger service.Logger, err error) {
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
  --force            Overwrite existing files or directories
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
  %s --force      # Overwrite any existing files
  %s --verbose    # Show detailed information
  %s --dry-run    # Simulate the operations
`, appName, appName, appName, appName, appName)
}

// displayVersion displays version information for the application
func displayVersion() {
	appName := filepath.Base(os.Args[0])
	fmt.Printf("%s version %s\n", appName, version)
}
