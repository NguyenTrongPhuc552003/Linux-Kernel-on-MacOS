// Package core provides core types, configuration, and context for elmos.
package core

import (
	"errors"
	"fmt"
)

// ErrCode represents error categories
type ErrCode int

const (
	// ErrCodeConfig indicates a configuration error
	ErrCodeConfig ErrCode = iota + 1
	// ErrCodeImage indicates a disk image error
	ErrCodeImage
	// ErrCodeRepo indicates a git repository error
	ErrCodeRepo
	// ErrCodeBuild indicates a kernel build error
	ErrCodeBuild
	// ErrCodeQEMU indicates a QEMU error
	ErrCodeQEMU
	// ErrCodeModule indicates a kernel module error
	ErrCodeModule
	// ErrCodeDependency indicates a missing dependency
	ErrCodeDependency
	// ErrCodePermission indicates insufficient permissions
	ErrCodePermission
)

// Error is a structured error type for elmos
type Error struct {
	Code    ErrCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.codeString(), e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.codeString(), e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) codeString() string {
	switch e.Code {
	case ErrCodeConfig:
		return "CONFIG"
	case ErrCodeImage:
		return "IMAGE"
	case ErrCodeRepo:
		return "REPO"
	case ErrCodeBuild:
		return "BUILD"
	case ErrCodeQEMU:
		return "QEMU"
	case ErrCodeModule:
		return "MODULE"
	case ErrCodeDependency:
		return "DEP"
	case ErrCodePermission:
		return "PERM"
	default:
		return "ERROR"
	}
}

// Error constructors

// ConfigError creates a configuration error
func ConfigError(msg string, cause error) *Error {
	return &Error{Code: ErrCodeConfig, Message: msg, Cause: cause}
}

// ImageError creates a disk image error
func ImageError(msg string, cause error) *Error {
	return &Error{Code: ErrCodeImage, Message: msg, Cause: cause}
}

// RepoError creates a repository error
func RepoError(msg string, cause error) *Error {
	return &Error{Code: ErrCodeRepo, Message: msg, Cause: cause}
}

// BuildError creates a build error
func BuildError(msg string, cause error) *Error {
	return &Error{Code: ErrCodeBuild, Message: msg, Cause: cause}
}

// QEMUError creates a QEMU error
func QEMUError(msg string, cause error) *Error {
	return &Error{Code: ErrCodeQEMU, Message: msg, Cause: cause}
}

// ModuleError creates a module error
func ModuleError(msg string, cause error) *Error {
	return &Error{Code: ErrCodeModule, Message: msg, Cause: cause}
}

// DependencyError creates a dependency error
func DependencyError(msg string, cause error) *Error {
	return &Error{Code: ErrCodeDependency, Message: msg, Cause: cause}
}

// PermissionError creates a permission error
func PermissionError(msg string, cause error) *Error {
	return &Error{Code: ErrCodePermission, Message: msg, Cause: cause}
}

// Common errors
var (
	ErrNotMounted     = errors.New("kernel volume is not mounted")
	ErrAlreadyMounted = errors.New("kernel volume is already mounted")
	ErrNoKernelDir    = errors.New("kernel directory does not exist")
	ErrNoConfig       = errors.New("kernel .config not found")
	ErrArchNotSet     = errors.New("target architecture not configured")
)
