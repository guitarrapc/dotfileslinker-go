package main

import (
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
		{name: "explicit false", args: []string{"--force=false"}, want: cliOptions{}},
		{name: "unknown option", args: []string{"--froce"}, wantError: "flag provided but not defined"},
		{name: "options are case sensitive", args: []string{"--FORCE"}, wantError: "flag provided but not defined"},
		{name: "legacy force value is invalid", args: []string{"--force=y"}, wantError: "invalid boolean value"},
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
