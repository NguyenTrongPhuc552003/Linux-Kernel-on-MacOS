// Package executor provides abstractions for executing shell commands.
package executor

import "context"

// Executor defines the interface for executing shell commands.
// This abstraction allows for easy mocking in tests and potential
// future implementations (e.g., remote execution, containerized execution).
type Executor interface {
	// Run executes a command and waits for it to complete.
	// stdout and stderr are connected to os.Stdout and os.Stderr.
	Run(ctx context.Context, cmd string, args ...string) error

	// RunWithEnv executes a command with custom environment variables.
	RunWithEnv(ctx context.Context, env []string, cmd string, args ...string) error

	// RunInDir executes a command in a specific working directory.
	RunInDir(ctx context.Context, dir string, cmd string, args ...string) error

	// RunWithEnvInDir executes a command with custom environment in a specific directory.
	RunWithEnvInDir(ctx context.Context, env []string, dir string, cmd string, args ...string) error

	// Output executes a command and returns its stdout.
	Output(ctx context.Context, cmd string, args ...string) ([]byte, error)

	// OutputWithEnv executes a command with custom environment and returns its stdout.
	OutputWithEnv(ctx context.Context, env []string, cmd string, args ...string) ([]byte, error)

	// LookPath searches for an executable in the system PATH.
	LookPath(cmd string) (string, error)

	// Exec replaces the current process with the specified command (syscall.Exec).
	// This is used for handing off to interactive programs like GDB.
	Exec(cmd string, args []string, env []string) error
}
