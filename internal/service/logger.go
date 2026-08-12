package service

import (
	"fmt"
	"io"
	"os"
)

// Logger provides an interface for logging operations.
type Logger interface {
	// Success logs a success message.
	Success(message string)

	// Error logs an error message.
	Error(message string)

	// Info logs an informational message.
	Info(message string)
	Infof(format string, arguments ...any)

	// Verbose logs a verbose message.
	Verbose(message string)
	Verbosef(format string, arguments ...any)
}

// NullLogger implements a logger that does nothing.
type NullLogger struct{}

// NewNullLogger creates a new instance of NullLogger.
func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

// Success does nothing for NullLogger.
func (nl *NullLogger) Success(message string) {}

// Error does nothing for NullLogger.
func (nl *NullLogger) Error(message string) {}

// Info does nothing for NullLogger.
func (nl *NullLogger) Info(message string)                   {}
func (nl *NullLogger) Infof(format string, arguments ...any) {}

// Verbose does nothing for NullLogger.
func (nl *NullLogger) Verbose(message string)                   {}
func (nl *NullLogger) Verbosef(format string, arguments ...any) {}

// ConsoleLogger implements a logger that writes to the console.
type ConsoleLogger struct {
	verbose     bool
	errorOutput io.Writer
}

// NewConsoleLogger creates a new instance of ConsoleLogger.
func NewConsoleLogger(verbose bool) *ConsoleLogger {
	return &ConsoleLogger{verbose: verbose, errorOutput: os.Stderr}
}

// Success logs a success message.
func (cl *ConsoleLogger) Success(message string) {
	writeSuccess(message)
}

// Error logs an error message.
func (cl *ConsoleLogger) Error(message string) {
	writeError(cl.errorOutput, message)
}

// Info logs an informational message.
func (cl *ConsoleLogger) Info(message string) {
	if cl.verbose {
		writeInfo(message)
	}
}

// Infof formats and logs an informational message only when verbose logging is enabled.
func (cl *ConsoleLogger) Infof(format string, arguments ...any) {
	if cl.verbose {
		writeInfof(format, arguments...)
	}
}

// Verbose logs a verbose message.
func (cl *ConsoleLogger) Verbose(message string) {
	if cl.verbose {
		writeVerbose(message)
	}
}

// Verbosef formats and logs a verbose message only when verbose logging is enabled.
func (cl *ConsoleLogger) Verbosef(format string, arguments ...any) {
	if cl.verbose {
		writeVerbosef(format, arguments...)
	}
}

// writeSuccess writes a success message.
func writeSuccess(msg string) {
	fmt.Println("\033[32m[o] " + msg + "\033[0m")
}

// writeError writes an error message to stderr or another configured error stream.
func writeError(output io.Writer, msg string) {
	if output == nil {
		output = os.Stderr
	}
	_, _ = fmt.Fprintln(output, "\033[31m[x] "+msg+"\033[0m")
}

// writeInfo writes an informational message.
func writeInfo(msg string) {
	fmt.Println("\033[36m[i] " + msg + "\033[0m")
}

func writeInfof(format string, arguments ...any) {
	fmt.Printf("\033[36m[i] "+format+"\033[0m\n", arguments...)
}

// writeVerbose writes a verbose message.
func writeVerbose(msg string) {
	fmt.Println("\033[33m[v] " + msg + "\033[0m")
}

func writeVerbosef(format string, arguments ...any) {
	fmt.Printf("\033[33m[v] "+format+"\033[0m\n", arguments...)
}
