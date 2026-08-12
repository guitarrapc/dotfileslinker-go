package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseOptions(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		want      cliOptions
		wantError string
	}{
		{name: "no options", args: nil, want: cliOptions{}},
		{name: "force flag", args: []string{"--force"}, want: cliOptions{forceOverwrite: true}},
		{name: "long options", args: []string{"--verbose", "--dry-run"}, want: cliOptions{verbose: true, dryRun: true}},
		{name: "short options", args: []string{"-v", "-d", "-h"}, want: cliOptions{verbose: true, dryRun: true, showHelp: true}},
		{name: "version", args: []string{"--version"}, want: cliOptions{showVersion: true}},
		{name: "repository root", args: []string{"--root", "../dotfiles"}, want: cliOptions{repositoryRoot: "../dotfiles"}},
		{name: "repository root equals form", args: []string{"--root=../dotfiles"}, want: cliOptions{repositoryRoot: "../dotfiles"}},
		{name: "explicit false", args: []string{"--force=false"}, want: cliOptions{}},
		{name: "unknown option", args: []string{"--froce"}, wantError: "flag provided but not defined"},
		{name: "options are case sensitive", args: []string{"--FORCE"}, wantError: "flag provided but not defined"},
		{name: "legacy force value is invalid", args: []string{"--force=y"}, wantError: "invalid boolean value"},
		{name: "missing repository root", args: []string{"--root"}, wantError: "flag needs an argument"},
		{name: "empty repository root", args: []string{"--root="}, wantError: "requires a non-empty path"},
		{name: "positional argument", args: []string{"repository"}, wantError: "unexpected argument"},
		{name: "argument after separator", args: []string{"--", "repository"}, wantError: "unexpected argument"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOptions(tt.args)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("parseOptions() error = %v, want error containing %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOptions() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseOptions() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolvePathOption(t *testing.T) {
	t.Run("command line takes precedence", func(t *testing.T) {
		t.Setenv("DOTFILES_ROOT", filepath.Join("environment", "dotfiles"))
		got, err := resolvePathOption(filepath.Join("option", "dotfiles"), "DOTFILES_ROOT", "default")
		if err != nil {
			t.Fatal(err)
		}
		want, err := filepath.Abs(filepath.Join("option", "dotfiles"))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("resolvePathOption() = %q, want %q", got, want)
		}
	})

	t.Run("environment takes precedence over default", func(t *testing.T) {
		t.Setenv("DOTFILES_ROOT", filepath.Join("environment", "dotfiles"))
		got, err := resolvePathOption("", "DOTFILES_ROOT", "default")
		if err != nil {
			t.Fatal(err)
		}
		want, err := filepath.Abs(filepath.Join("environment", "dotfiles"))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("resolvePathOption() = %q, want %q", got, want)
		}
	})

	t.Run("default is used", func(t *testing.T) {
		t.Setenv("DOTFILES_ROOT", "")
		got, err := resolvePathOption("", "DOTFILES_ROOT", filepath.Join("default", "dotfiles"))
		if err != nil {
			t.Fatal(err)
		}
		want, err := filepath.Abs(filepath.Join("default", "dotfiles"))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("resolvePathOption() = %q, want %q", got, want)
		}
	})
}
