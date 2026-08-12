package service

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestConsoleLoggerUsesStderrForErrors(t *testing.T) {
	logger := NewConsoleLogger(false)
	if logger.errorOutput != os.Stderr {
		t.Fatal("NewConsoleLogger() does not configure stderr as the error output")
	}

	var output bytes.Buffer
	logger.errorOutput = &output
	logger.Error("operation failed")

	got := output.String()
	if !strings.Contains(got, "[x] operation failed") {
		t.Fatalf("error output = %q", got)
	}
}
