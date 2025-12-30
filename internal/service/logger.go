package service

import (
	"fmt"
)

// Logger provides an interface for logging operations.
type Logger interface {
	// Success logs a success message.
	Success(message string)

	// Error logs an error message.
	Error(message string)

	// Info logs an informational message.
	Info(message string)

	// Verbose logs a verbose message.
	Verbose(message string)
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
func (nl *NullLogger) Info(message string) {}

// Verbose does nothing for NullLogger.
func (nl *NullLogger) Verbose(message string) {}

// ConsoleLogger implements a logger that writes to the console.
type ConsoleLogger struct {
	verbose bool
}

// NewConsoleLogger creates a new instance of ConsoleLogger.
func NewConsoleLogger(verbose bool) *ConsoleLogger {
	return &ConsoleLogger{verbose: verbose}
}

// Success logs a success message.
func (cl *ConsoleLogger) Success(message string) {
	writeSuccess(message)
}

// Error logs an error message.
func (cl *ConsoleLogger) Error(message string) {
	writeError(message)
}

// Info logs an informational message.
func (cl *ConsoleLogger) Info(message string) {
	if cl.verbose {
		writeInfo(message)
	}
}

// Verbose logs a verbose message.
func (cl *ConsoleLogger) Verbose(message string) {
	if cl.verbose {
		writeVerbose(message)
	}
}

// writeSuccess writes a success message.
func writeSuccess(msg string) {
	fmt.Println("\033[32m[o] " + msg + "\033[0m")
}

// writeError writes an error message.
func writeError(msg string) {
	fmt.Println("\033[31m[x] " + msg + "\033[0m")
}

// writeInfo writes an informational message.
func writeInfo(msg string) {
	fmt.Println("\033[36m[i] " + msg + "\033[0m")
}

// writeVerbose writes a verbose message.
func writeVerbose(msg string) {
	fmt.Println("\033[33m[v] " + msg + "\033[0m")
}
