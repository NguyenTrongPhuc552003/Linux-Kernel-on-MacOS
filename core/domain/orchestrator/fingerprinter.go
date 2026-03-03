// Package orchestrator provides build pipeline orchestration for ELMOS.
// This file implements content-hash fingerprinting for build artifact caching.
package orchestrator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BuildFingerprinter computes deterministic content-hash fingerprints for build inputs.
// The same inputs always produce the same fingerprint, enabling precise cache lookups.
type BuildFingerprinter struct {
	indexPath string
}

// CacheIndexEntry records a single cached build.
type CacheIndexEntry struct {
	Fingerprint string    `json:"fingerprint"`
	Component   string    `json:"component"` // "kernel", "bootloader", "rootfs"
	Machine     string    `json:"machine"`   // board name
	CreatedAt   time.Time `json:"created_at"`
	Duration    int64     `json:"duration_sec"` // build duration
	Success     bool      `json:"success"`
	Artifacts   []string  `json:"artifacts"`  // relative paths under cache dir
	SizeBytes   int64     `json:"size_bytes"` // total artifact size
}

// CacheIndex is the in-memory representation of the build cache index.
type CacheIndex struct {
	Entries []CacheIndexEntry `json:"entries"`
}

// KernelFingerprintInput represents inputs to the kernel fingerprint.
type KernelFingerprintInput struct {
	Version       string   // Kernel version or git tag
	Arch          string   // ARCH= value
	Defconfig     string   // defconfig name or content hash
	PatchHashes   []string // SHA256 hashes of applied patches
	GitRevision   string   // kernel source git HEAD revision
	ConfigOptions []string // Additional CONFIG_* options
}

// BootloaderFingerprintInput represents inputs to the bootloader fingerprint.
type BootloaderFingerprintInput struct {
	UBootVersion   string            // U-Boot version/tag
	Defconfig      string            // board defconfig name
	PatchHashes    []string          // SHA256 hashes of applied patches
	FirmwareHashes map[string]string // firmware blob name → SHA256
	CrossCompile   string            // cross-compile prefix
}

// RootfsFingerprintInput represents inputs for rootfs fingerprinting.
type RootfsFingerprintInput struct {
	Distribution string   // e.g. "debian"
	Release      string   // e.g. "bookworm"
	Packages     []string // sorted package list
	ScriptHash   string   // hash of post-build script (empty if none)
}

// NewBuildFingerprinter creates a new BuildFingerprinter.
// indexPath is where the cache index JSON is stored.
func NewBuildFingerprinter(indexPath string) *BuildFingerprinter {
	return &BuildFingerprinter{
		indexPath: indexPath,
	}
}

// FingerprintKernel generates a deterministic fingerprint from kernel build inputs.
func (bf *BuildFingerprinter) FingerprintKernel(input KernelFingerprintInput) (string, error) {
	h := sha256.New()

	// Sort patch hashes for determinism
	patchHashes := make([]string, len(input.PatchHashes))
	copy(patchHashes, input.PatchHashes)
	sort.Strings(patchHashes)

	// Sort config options
	configOpts := make([]string, len(input.ConfigOptions))
	copy(configOpts, input.ConfigOptions)
	sort.Strings(configOpts)

	// Build structured input for hashing
	payload := map[string]interface{}{
		"type":           "kernel",
		"version":        input.Version,
		"arch":           input.Arch,
		"defconfig":      input.Defconfig,
		"patches":        patchHashes,
		"git_revision":   input.GitRevision,
		"config_options": configOpts,
	}

	return bf.hashPayload(h, payload)
}

// FingerprintBootloader generates a deterministic fingerprint from U-Boot build inputs.
func (bf *BuildFingerprinter) FingerprintBootloader(input BootloaderFingerprintInput) (string, error) {
	h := sha256.New()

	// Sort patch hashes
	patchHashes := make([]string, len(input.PatchHashes))
	copy(patchHashes, input.PatchHashes)
	sort.Strings(patchHashes)

	// Sort firmware hashes by blob name
	firmwareHashes := make(map[string]string)
	for k, v := range input.FirmwareHashes {
		firmwareHashes[k] = v
	}

	payload := map[string]interface{}{
		"type":            "bootloader",
		"uboot_version":   input.UBootVersion,
		"defconfig":       input.Defconfig,
		"patches":         patchHashes,
		"firmware_hashes": firmwareHashes,
		"cross_compile":   input.CrossCompile,
	}

	return bf.hashPayload(h, payload)
}

// FingerprintRootfs generates a deterministic fingerprint from rootfs inputs.
func (bf *BuildFingerprinter) FingerprintRootfs(input RootfsFingerprintInput) (string, error) {
	h := sha256.New()

	// Sort packages for determinism
	packages := make([]string, len(input.Packages))
	copy(packages, input.Packages)
	sort.Strings(packages)

	payload := map[string]interface{}{
		"type":         "rootfs",
		"distribution": input.Distribution,
		"release":      input.Release,
		"packages":     packages,
		"script_hash":  input.ScriptHash,
	}

	return bf.hashPayload(h, payload)
}

// HashFile computes the SHA256 hash of a file.
// Useful for hashing patch files and config files.
func (bf *BuildFingerprinter) HashFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash file %s: %w", filePath, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashDirectory computes a combined SHA256 of all files in a directory.
// File paths are sorted for determinism. Used for hashing patch directories.
func (bf *BuildFingerprinter) HashDirectory(dirPath string) (string, error) {
	h := sha256.New()
	var filePaths []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			filePaths = append(filePaths, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	// Sort for determinism
	sort.Strings(filePaths)

	for _, path := range filePaths {
		f, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("failed to open %s: %w", path, err)
		}
		relPath, _ := filepath.Rel(dirPath, path)
		// Include relative path in hash to distinguish same-content files at different paths
		_, _ = fmt.Fprintf(h, "%s:", relPath)
		_, _ = io.Copy(h, f)
		_ = f.Close()
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// GetGitRevision returns the current HEAD commit hash of a git repository.
func (bf *BuildFingerprinter) GetGitRevision(repoPath string) (string, error) {
	revPath := filepath.Join(repoPath, ".git", "HEAD")
	data, err := os.ReadFile(revPath)
	if err != nil {
		return "", fmt.Errorf("failed to read git HEAD at %s: %w", repoPath, err)
	}

	ref := string(data)
	// If HEAD is a symbolic ref (e.g., refs/heads/main), resolve it
	const refPrefix = "ref: "
	if len(ref) > len(refPrefix) && ref[:len(refPrefix)] == refPrefix {
		refFile := filepath.Join(repoPath, ".git", ref[len(refPrefix):len(ref)-1])
		resolvedData, err := os.ReadFile(refFile)
		if err != nil {
			return "", fmt.Errorf("failed to resolve git ref: %w", err)
		}
		return string(resolvedData[:40]), nil
	}

	// Detached HEAD - ref is already a hash
	if len(ref) >= 40 {
		return ref[:40], nil
	}
	return "", fmt.Errorf("unexpected git HEAD format: %s", ref)
}

// --- Cache Index Management ---

// LoadIndex reads the cache index from disk.
// Returns an empty index if the file doesn't exist.
func (bf *BuildFingerprinter) LoadIndex() (*CacheIndex, error) {
	data, err := os.ReadFile(bf.indexPath)
	if os.IsNotExist(err) {
		return &CacheIndex{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read cache index: %w", err)
	}

	var index CacheIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse cache index: %w", err)
	}
	return &index, nil
}

// SaveIndex writes the cache index to disk.
func (bf *BuildFingerprinter) SaveIndex(index *CacheIndex) error {
	if err := os.MkdirAll(filepath.Dir(bf.indexPath), 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache index: %w", err)
	}

	return os.WriteFile(bf.indexPath, data, 0644)
}

// RecordBuild adds or updates a cache index entry for a completed build.
func (bf *BuildFingerprinter) RecordBuild(entry CacheIndexEntry) error {
	index, err := bf.LoadIndex()
	if err != nil {
		index = &CacheIndex{}
	}

	// Replace existing entry with same fingerprint or append
	found := false
	for i, existing := range index.Entries {
		if existing.Fingerprint == entry.Fingerprint && existing.Component == entry.Component {
			index.Entries[i] = entry
			found = true
			break
		}
	}
	if !found {
		index.Entries = append(index.Entries, entry)
	}

	return bf.SaveIndex(index)
}

// LookupEntry finds a cache entry by fingerprint and component.
// Returns nil if no matching entry exists.
func (bf *BuildFingerprinter) LookupEntry(fingerprint, component string) *CacheIndexEntry {
	index, err := bf.LoadIndex()
	if err != nil {
		return nil
	}

	for i, entry := range index.Entries {
		if entry.Fingerprint == fingerprint && entry.Component == component {
			return &index.Entries[i]
		}
	}
	return nil
}

// PruneStaleEntries removes entries older than maxAge from the index.
func (bf *BuildFingerprinter) PruneStaleEntries(maxAge time.Duration) (int, error) {
	index, err := bf.LoadIndex()
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-maxAge)
	var kept []CacheIndexEntry
	pruned := 0

	for _, entry := range index.Entries {
		if entry.CreatedAt.After(cutoff) {
			kept = append(kept, entry)
		} else {
			pruned++
		}
	}

	index.Entries = kept
	return pruned, bf.SaveIndex(index)
}

// --- Internal helpers ---

// hashPayload converts a map to sorted JSON and returns its SHA256 hex digest.
func (bf *BuildFingerprinter) hashPayload(h io.Writer, payload map[string]interface{}) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fingerprint payload: %w", err)
	}

	_, _ = h.Write(data)

	// h must be a hash.Hash; use type assertion for Sum
	type hasher interface {
		Sum(b []byte) []byte
	}
	if hh, ok := h.(hasher); ok {
		return hex.EncodeToString(hh.Sum(nil)), nil
	}

	return "", fmt.Errorf("internal error: writer does not implement hash.Hash")
}
