// Package platform provides OS-specific abstractions for ELMOS operations.
// This file implements runtime OS detection and the singleton accessor.
package platform

import (
	"os"
	"runtime"
	"sync"

	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
)

var (
	detectOnce      sync.Once
	defaultPlatform Platform
)

// Current returns the Platform implementation for the current OS.
// The result is cached after the first call. Uses a no-op executor by default;
// call CurrentWithExecutor before any platform operations that run shell commands.
func Current() Platform {
	detectOnce.Do(func() {
		defaultPlatform = detect(nil)
	})
	return defaultPlatform
}

// CurrentWithExecutor returns a freshly constructed Platform implementation
// that uses the provided executor for all shell commands.
// This should be called once during app initialisation and the result stored
// on the application context.
func CurrentWithExecutor(exec executor.Executor) Platform {
	return detect(exec)
}

// detect constructs the right Platform impl for runtime.GOOS.
func detect(exec executor.Executor) Platform {
	switch runtime.GOOS {
	case "darwin":
		return newDarwinPlatform(exec)
	case "linux":
		p := newLinuxPlatform(exec)
		// Detect OrbStack VM: macOS paths are accessible via virtiofs but
		// hdiutil/diskutil do not exist inside the guest. The Linux platform
		// handles this correctly (truncate + losetup + mount).
		if isOrbStack() {
			p.orbstack = true
		}
		return p
	case "windows":
		return newWindowsPlatform(exec)
	default:
		// Best-effort fallback: treat as generic Linux.
		return newLinuxPlatform(exec)
	}
}

// isOrbStack returns true when running inside an OrbStack Linux VM.
// OrbStack installs its helper binaries in /opt/orbstack-guest/.
func isOrbStack() bool {
	_, err := os.Stat("/opt/orbstack-guest")
	return err == nil
}
