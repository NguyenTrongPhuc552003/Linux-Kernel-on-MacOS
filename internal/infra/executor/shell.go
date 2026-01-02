// Package executor provides abstractions for executing shell commands.
package executor

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

// ShellExecutor implements Executor using os/exec.
type ShellExecutor struct {
	// Stdout is the writer for command stdout. Defaults to os.Stdout.
	Stdout *os.File
	// Stderr is the writer for command stderr. Defaults to os.Stderr.
	Stderr *os.File
	// Stdin is the reader for command stdin. Defaults to os.Stdin.
	Stdin *os.File
}

// NewShellExecutor creates a new ShellExecutor with default stdio.
func NewShellExecutor() *ShellExecutor {
	return &ShellExecutor{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
	}
}

// Run executes a command and waits for it to complete.
func (e *ShellExecutor) Run(ctx context.Context, cmd string, args ...string) error {
	return e.RunWithEnvInDir(ctx, nil, "", cmd, args...)
}

// RunWithEnv executes a command with custom environment variables.
func (e *ShellExecutor) RunWithEnv(ctx context.Context, env []string, cmd string, args ...string) error {
	return e.RunWithEnvInDir(ctx, env, "", cmd, args...)
}

// RunInDir executes a command in a specific working directory.
func (e *ShellExecutor) RunInDir(ctx context.Context, dir string, cmd string, args ...string) error {
	return e.RunWithEnvInDir(ctx, nil, dir, cmd, args...)
}

// RunWithEnvInDir executes a command with custom environment in a specific directory.
func (e *ShellExecutor) RunWithEnvInDir(ctx context.Context, env []string, dir string, cmd string, args ...string) error {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Stdout = e.Stdout
	c.Stderr = e.Stderr
	c.Stdin = e.Stdin

	if dir != "" {
		c.Dir = dir
	}

	if len(env) > 0 {
		c.Env = env
	}

	return c.Run()
}

// Output executes a command and returns its stdout.
func (e *ShellExecutor) Output(ctx context.Context, cmd string, args ...string) ([]byte, error) {
	return e.OutputWithEnv(ctx, nil, cmd, args...)
}

// OutputWithEnv executes a command with custom environment and returns its stdout.
func (e *ShellExecutor) OutputWithEnv(ctx context.Context, env []string, cmd string, args ...string) ([]byte, error) {
	c := exec.CommandContext(ctx, cmd, args...)
	if len(env) > 0 {
		c.Env = env
	}
	return c.Output()
}

// LookPath searches for an executable in the system PATH.
func (e *ShellExecutor) LookPath(cmd string) (string, error) {
	return exec.LookPath(cmd)
}

// Exec replaces the current process with the specified command.
func (e *ShellExecutor) Exec(cmd string, args []string, env []string) error {
	return syscall.Exec(cmd, args, env)
}

// Ensure ShellExecutor implements Executor.
var _ Executor = (*ShellExecutor)(nil)
