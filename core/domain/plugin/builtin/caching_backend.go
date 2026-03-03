// Package builtin provides built-in plugins for the ELMOS system.
// This file implements the caching backend plugin that integrates with
// the BuildFingerprinter to skip redundant builds.
package builtin

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
	"github.com/NguyenTrongPhuc552003/elmos/core/domain/orchestrator"
	"github.com/NguyenTrongPhuc552003/elmos/core/plugin"
)

// CachingBackendPlugin wraps BuildFingerprinter to check and populate the
// build artifact cache around kernel, bootloader, and rootfs builds.
type CachingBackendPlugin struct {
	fp       *orchestrator.BuildFingerprinter
	cacheDir string // root artifact store: ~/.elmos/build-cache/
}

// NewCachingBackendPlugin creates a new CachingBackend plugin.
// cacheDir is typically ~/.elmos/build-cache/.
func NewCachingBackendPlugin(cacheDir string) (*CachingBackendPlugin, error) {
	fp := orchestrator.NewBuildFingerprinter(filepath.Join(cacheDir, "index.json"))
	return &CachingBackendPlugin{fp: fp, cacheDir: cacheDir}, nil
}

// Name returns the plugin name.
func (c *CachingBackendPlugin) Name() string { return "caching-backend" }

// Version returns the plugin version.
func (c *CachingBackendPlugin) Version() string { return "1.0.0" }

// Description returns the plugin description.
func (c *CachingBackendPlugin) Description() string {
	return "Build artifact caching via SHA256 content fingerprinting"
}

// Init initialises the plugin — creates the cache directory if needed.
func (c *CachingBackendPlugin) Init(_ *elcontext.Context, cfg map[string]interface{}) error {
	if cfg != nil {
		if dir, ok := cfg["cache_dir"].(string); ok && dir != "" {
			c.cacheDir = dir
			c.fp = orchestrator.NewBuildFingerprinter(filepath.Join(dir, "index.json"))
		}
	}
	return os.MkdirAll(c.cacheDir, 0755)
}

// Validate checks that the cache directory is writable.
func (c *CachingBackendPlugin) Validate() error {
	testFile := filepath.Join(c.cacheDir, ".write-test")
	if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
		return fmt.Errorf("caching-backend: cache dir %q is not writable: %w", c.cacheDir, err)
	}
	return os.Remove(testFile)
}

// Cleanup is a no-op for the caching backend.
func (c *CachingBackendPlugin) Cleanup() error { return nil }

// Hooks registers pre and post hooks for kernel, bootloader, and rootfs.
func (c *CachingBackendPlugin) Hooks() []plugin.HookRegistration {
	return []plugin.HookRegistration{
		{Event: plugin.PreKernelBuild, Handler: c.onPreBuild("kernel"), Priority: 9},
		{Event: plugin.PostKernelBuild, Handler: c.onPostBuild("kernel"), Priority: 1},
		{Event: plugin.PreBootloaderBuild, Handler: c.onPreBuild("bootloader"), Priority: 9},
		{Event: plugin.PostBootloaderBuild, Handler: c.onPostBuild("bootloader"), Priority: 1},
		{Event: plugin.PreRootfsCreate, Handler: c.onPreBuild("rootfs"), Priority: 9},
		{Event: plugin.PostRootfsCreate, Handler: c.onPostBuild("rootfs"), Priority: 1},
	}
}

// onPreBuild returns a hook that checks the cache for component.
// If a cache hit is found, it sets evt.Metadata["cache.hit"] = true so
// the actual builder plugin can skip its work.
func (c *CachingBackendPlugin) onPreBuild(component string) plugin.HookFunc {
	return func(_ context.Context, evt *plugin.Event) error {
		fp, ok := evt.Metadata["build.fingerprint"].(string)
		if !ok || fp == "" {
			return nil // fingerprint not computed yet — skip cache check
		}
		entry := c.fp.LookupEntry(fp, component)
		if entry != nil && entry.Success {
			evt.Metadata["cache.hit"] = true
			evt.Metadata["cache.artifacts"] = entry.Artifacts
		}
		return nil
	}
}

// onPostBuild returns a hook that records a successful build in the cache index
// and copies artifacts into the content-addressed artifact store.
func (c *CachingBackendPlugin) onPostBuild(component string) plugin.HookFunc {
	return func(_ context.Context, evt *plugin.Event) error {
		fp, ok := evt.Metadata["build.fingerprint"].(string)
		if !ok || fp == "" {
			return nil // no fingerprint — nothing to cache
		}
		if hit, _ := evt.Metadata["cache.hit"].(bool); hit {
			return nil // already came from cache; don't re-record
		}

		artifacts, _ := evt.Metadata["build.artifacts"].([]string)
		storedPaths, err := c.storeArtifacts(fp, component, artifacts)
		if err != nil {
			return fmt.Errorf("caching-backend: failed to store artifacts: %w", err)
		}

		duration, _ := evt.Metadata[plugin.MetadataDuration].(int64)
		return c.fp.RecordBuild(orchestrator.CacheIndexEntry{
			Fingerprint: fp,
			Component:   component,
			CreatedAt:   time.Now(),
			Duration:    duration,
			Success:     true,
			Artifacts:   storedPaths,
		})
	}
}

// storeArtifacts copies source paths into the content-addressed cache store.
// Destination: <cacheDir>/<component>/<fingerprint[:8]>/<filename>
func (c *CachingBackendPlugin) storeArtifacts(fp, component string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	destDir := filepath.Join(c.cacheDir, component, fp[:8])
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}
	stored := make([]string, 0, len(paths))
	for _, src := range paths {
		dest := filepath.Join(destDir, filepath.Base(src))
		if err := copyFile(src, dest); err != nil {
			return nil, fmt.Errorf("failed to cache %s: %w", filepath.Base(src), err)
		}
		stored = append(stored, dest)
	}
	return stored, nil
}

// RestoreArtifacts copies cached artifacts into destDir.
// Returns false if no cache entry exists for the given fingerprint+component.
func (c *CachingBackendPlugin) RestoreArtifacts(fp, component, destDir string) (bool, error) {
	entry := c.fp.LookupEntry(fp, component)
	if entry == nil {
		return false, nil
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create dest dir: %w", err)
	}
	for _, src := range entry.Artifacts {
		dest := filepath.Join(destDir, filepath.Base(src))
		if err := copyFile(src, dest); err != nil {
			return false, fmt.Errorf("failed to restore %s: %w", filepath.Base(src), err)
		}
	}
	return true, nil
}

// Fingerprinter returns the underlying BuildFingerprinter for direct access.
func (c *CachingBackendPlugin) Fingerprinter() *orchestrator.BuildFingerprinter {
	return c.fp
}

// copyFile copies src to dst, creating dst if it does not exist.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
