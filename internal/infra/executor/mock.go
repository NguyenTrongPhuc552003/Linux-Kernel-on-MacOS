// Package executor provides abstractions for executing shell commands.
package executor

import (
	"context"
	"fmt"
)

// CommandCall records a single command execution for verification in tests.
type CommandCall struct {
	Cmd  string
	Args []string
	Env  []string
	Dir  string
}

// MockExecutor is a test double for Executor that records calls and returns configured responses.
type MockExecutor struct {
	// Calls records all command executions in order.
	Calls []CommandCall

	// RunError is returned by Run methods when set.
	RunError error

	// OutputResponses maps command names to their mock output.
	OutputResponses map[string][]byte

	// OutputErrors maps command names to their mock errors.
	OutputErrors map[string]error

	// LookPathResponses maps command names to their mock paths.
	LookPathResponses map[string]string

	// LookPathErrors maps command names to their mock errors.
	LookPathErrors map[string]error

	// ExecCalled is set to true when Exec is called.
	ExecCalled bool

	// ExecError is returned by Exec when set.
	ExecError error
}

// NewMockExecutor creates a new MockExecutor with empty response maps.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Calls:             make([]CommandCall, 0),
		OutputResponses:   make(map[string][]byte),
		OutputErrors:      make(map[string]error),
		LookPathResponses: make(map[string]string),
		LookPathErrors:    make(map[string]error),
	}
}

// Run records the command execution.
func (m *MockExecutor) Run(ctx context.Context, cmd string, args ...string) error {
	m.Calls = append(m.Calls, CommandCall{Cmd: cmd, Args: args})
	return m.RunError
}

// RunWithEnv records the command execution with environment.
func (m *MockExecutor) RunWithEnv(ctx context.Context, env []string, cmd string, args ...string) error {
	m.Calls = append(m.Calls, CommandCall{Cmd: cmd, Args: args, Env: env})
	return m.RunError
}

// RunInDir records the command execution with directory.
func (m *MockExecutor) RunInDir(ctx context.Context, dir string, cmd string, args ...string) error {
	m.Calls = append(m.Calls, CommandCall{Cmd: cmd, Args: args, Dir: dir})
	return m.RunError
}

// RunWithEnvInDir records the command execution with environment and directory.
func (m *MockExecutor) RunWithEnvInDir(ctx context.Context, env []string, dir string, cmd string, args ...string) error {
	m.Calls = append(m.Calls, CommandCall{Cmd: cmd, Args: args, Env: env, Dir: dir})
	return m.RunError
}

// Output returns the configured mock output for the command.
func (m *MockExecutor) Output(ctx context.Context, cmd string, args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, CommandCall{Cmd: cmd, Args: args})

	if err, ok := m.OutputErrors[cmd]; ok {
		return nil, err
	}
	if out, ok := m.OutputResponses[cmd]; ok {
		return out, nil
	}
	return nil, nil
}

// OutputWithEnv returns the configured mock output for the command.
func (m *MockExecutor) OutputWithEnv(ctx context.Context, env []string, cmd string, args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, CommandCall{Cmd: cmd, Args: args, Env: env})

	if err, ok := m.OutputErrors[cmd]; ok {
		return nil, err
	}
	if out, ok := m.OutputResponses[cmd]; ok {
		return out, nil
	}
	return nil, nil
}

// LookPath returns the configured mock path for the command.
func (m *MockExecutor) LookPath(cmd string) (string, error) {
	if err, ok := m.LookPathErrors[cmd]; ok {
		return "", err
	}
	if path, ok := m.LookPathResponses[cmd]; ok {
		return path, nil
	}
	return "", fmt.Errorf("executable file not found in $PATH: %s", cmd)
}

// Exec records that Exec was called and returns the configured error.
func (m *MockExecutor) Exec(cmd string, args []string, env []string) error {
	m.ExecCalled = true
	m.Calls = append(m.Calls, CommandCall{Cmd: cmd, Args: args, Env: env})
	return m.ExecError
}

// Reset clears all recorded calls.
func (m *MockExecutor) Reset() {
	m.Calls = make([]CommandCall, 0)
	m.ExecCalled = false
}

// Ensure MockExecutor implements Executor.
var _ Executor = (*MockExecutor)(nil)
