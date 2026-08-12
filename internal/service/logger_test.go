package service

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

type formattingProbe struct {
	calls int
}

func (p *formattingProbe) String() string {
	p.calls++
	return "formatted"
}

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

func TestDisabledConsoleLoggerDoesNotFormatMessages(t *testing.T) {
	logger := NewConsoleLogger(false)
	probe := &formattingProbe{}

	logger.Infof("info: %s", probe)
	logger.Verbosef("verbose: %s", probe)

	if probe.calls != 0 {
		t.Fatalf("disabled logger formatted messages %d times", probe.calls)
	}
}
