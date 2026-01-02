// Package homebrew provides utilities for resolving Homebrew package paths.
package homebrew

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
)

// Resolver provides methods to resolve Homebrew package paths.
type Resolver struct {
	exec executor.Executor
	// cache stores resolved prefixes to avoid repeated brew calls
	cache map[string]string
}

// NewResolver creates a new Homebrew Resolver.
func NewResolver(exec executor.Executor) *Resolver {
	return &Resolver{
		exec:  exec,
		cache: make(map[string]string),
	}
}

// GetPrefix returns the installation prefix for a Homebrew package.
func (r *Resolver) GetPrefix(pkg string) string {
	if cached, ok := r.cache[pkg]; ok {
		return cached
	}

	out, err := r.exec.Output(context.Background(), "brew", "--prefix", pkg)
	if err != nil {
		return ""
	}

	prefix := strings.TrimSpace(string(out))
	r.cache[pkg] = prefix
	return prefix
}

// GetBin returns the bin directory path for a Homebrew package.
func (r *Resolver) GetBin(pkg string) string {
	prefix := r.GetPrefix(pkg)
	if prefix == "" {
		return ""
	}
	return filepath.Join(prefix, "bin")
}

// GetSbin returns the sbin directory path for a Homebrew package.
func (r *Resolver) GetSbin(pkg string) string {
	prefix := r.GetPrefix(pkg)
	if prefix == "" {
		return ""
	}
	return filepath.Join(prefix, "sbin")
}

// GetInclude returns the include directory path for a Homebrew package.
func (r *Resolver) GetInclude(pkg string) string {
	prefix := r.GetPrefix(pkg)
	if prefix == "" {
		return ""
	}
	return filepath.Join(prefix, "include")
}

// GetLib returns the lib directory path for a Homebrew package.
func (r *Resolver) GetLib(pkg string) string {
	prefix := r.GetPrefix(pkg)
	if prefix == "" {
		return ""
	}
	return filepath.Join(prefix, "lib")
}

// GetLibexecBin returns the libexec/gnubin path for GNU tools.
// This is used for tools like gnu-sed and coreutils.
func (r *Resolver) GetLibexecBin(pkg string) string {
	prefix := r.GetPrefix(pkg)
	if prefix == "" {
		return ""
	}
	return filepath.Join(prefix, "libexec", "gnubin")
}

// ListInstalled returns a list of installed Homebrew formulae.
func (r *Resolver) ListInstalled() ([]string, error) {
	out, err := r.exec.Output(context.Background(), "brew", "list", "--formulae")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines, nil
}

// ListTaps returns a list of tapped Homebrew repositories.
func (r *Resolver) ListTaps() ([]string, error) {
	out, err := r.exec.Output(context.Background(), "brew", "tap")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines, nil
}

// IsInstalled checks if a Homebrew package is installed.
func (r *Resolver) IsInstalled(pkg string) bool {
	installed, err := r.ListInstalled()
	if err != nil {
		return false
	}

	for _, p := range installed {
		if p == pkg {
			return true
		}
	}
	return false
}

// IsTapped checks if a Homebrew tap is tapped.
func (r *Resolver) IsTapped(tap string) bool {
	taps, err := r.ListTaps()
	if err != nil {
		return false
	}

	for _, t := range taps {
		if t == tap {
			return true
		}
	}
	return false
}

// ClearCache clears the prefix cache.
func (r *Resolver) ClearCache() {
	r.cache = make(map[string]string)
}
