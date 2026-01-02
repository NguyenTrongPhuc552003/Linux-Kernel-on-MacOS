// Package ui provides console output helpers for elmos.
package ui

import (
	"fmt"
	"os"
)

// Printer provides formatted console output.
type Printer struct{}

// NewPrinter creates a new Printer.
func NewPrinter() *Printer {
	return &Printer{}
}

// Success prints a success message with a checkmark.
func (p *Printer) Success(format string, args ...interface{}) {
	fmt.Println(SuccessStyle.Render(fmt.Sprintf("✓ "+format, args...)))
}

// Error prints an error message with an X.
func (p *Printer) Error(format string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, ErrorStyle.Render(fmt.Sprintf("✗ "+format, args...)))
}

// Warn prints a warning message with a warning sign.
func (p *Printer) Warn(format string, args ...interface{}) {
	fmt.Println(WarnStyle.Render(fmt.Sprintf("⚠ "+format, args...)))
}

// Info prints an info message with an info sign.
func (p *Printer) Info(format string, args ...interface{}) {
	fmt.Println(InfoStyle.Render(fmt.Sprintf("ℹ "+format, args...)))
}

// Step prints a step message with an arrow.
func (p *Printer) Step(format string, args ...interface{}) {
	fmt.Println(AccentStyle.Render(fmt.Sprintf("→ "+format, args...)))
}

// Print prints a plain message.
func (p *Printer) Print(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// Global printer instance for convenience.
var defaultPrinter = NewPrinter()

// PrintSuccess prints a success message.
func PrintSuccess(format string, args ...interface{}) {
	defaultPrinter.Success(format, args...)
}

// PrintError prints an error message.
func PrintError(format string, args ...interface{}) {
	defaultPrinter.Error(format, args...)
}

// PrintWarn prints a warning message.
func PrintWarn(format string, args ...interface{}) {
	defaultPrinter.Warn(format, args...)
}

// PrintInfo prints an info message.
func PrintInfo(format string, args ...interface{}) {
	defaultPrinter.Info(format, args...)
}

// PrintStep prints a step message.
func PrintStep(format string, args ...interface{}) {
	defaultPrinter.Step(format, args...)
}
