// Package bsp provides board support package management for ELMOS.
// It handles firmware blob discovery, downloading, caching, and validation.
package bsp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FirmwareMgr manages firmware blob downloads and caching.
// Blobs are stored in ~/.elmos/bsp-cache/firmware/<soc>/<blob-name>
type FirmwareMgr struct {
	cacheDir   string
	httpClient *http.Client
}

// NewFirmwareMgr creates a new FirmwareMgr with the given cache directory.
func NewFirmwareMgr(cacheDir string) *FirmwareMgr {
	return &FirmwareMgr{
		cacheDir: filepath.Join(cacheDir, "firmware"),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// GetBlobPath returns the local path for a firmware blob.
// Creates SoC-specific subdirectory under cacheDir.
func (fm *FirmwareMgr) GetBlobPath(soc, blobName string) string {
	return filepath.Join(fm.cacheDir, soc, blobName)
}

// IsCached returns true if the blob is already cached locally.
func (fm *FirmwareMgr) IsCached(soc, blobName string) bool {
	path := fm.GetBlobPath(soc, blobName)
	_, err := os.Stat(path)
	return err == nil
}

// EnsureBlob ensures a firmware blob is available locally.
// Downloads from URL if not cached, validates checksum if provided.
func (fm *FirmwareMgr) EnsureBlob(soc, blobName, sourceURL, expectedChecksum string) (string, error) {
	destPath := fm.GetBlobPath(soc, blobName)

	// Return from cache if exists and checksum is valid
	if fm.IsCached(soc, blobName) {
		if expectedChecksum == "" {
			return destPath, nil
		}
		if err := fm.ValidateChecksum(destPath, expectedChecksum); err == nil {
			return destPath, nil
		}
		// Checksum mismatch - re-download
	}

	// Create SoC directory
	socDir := filepath.Join(fm.cacheDir, soc)
	if err := os.MkdirAll(socDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create firmware cache dir for %s: %w", soc, err)
	}

	// Download blob
	if err := fm.downloadBlob(sourceURL, destPath); err != nil {
		return "", fmt.Errorf("failed to download firmware blob %s: %w", blobName, err)
	}

	// Validate checksum after download
	if expectedChecksum != "" {
		if err := fm.ValidateChecksum(destPath, expectedChecksum); err != nil {
			_ = os.Remove(destPath)
			return "", fmt.Errorf("firmware blob %s checksum mismatch: %w", blobName, err)
		}
	}

	return destPath, nil
}

// DownloadBlobs downloads all blobs for a given machine concurrently.
// Returns a map of blob name → local path for all successfully downloaded blobs.
func (fm *FirmwareMgr) DownloadBlobs(blobs []BlobSpec) (map[string]string, error) {
	type result struct {
		blobName string
		path     string
		err      error
	}

	results := make(chan result, len(blobs))

	// Limit concurrent downloads
	sem := make(chan struct{}, 3)

	var wg sync.WaitGroup
	for _, blob := range blobs {
		wg.Add(1)
		blob := blob // capture
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			path, err := fm.EnsureBlob(blob.SoC, blob.Name, blob.URL, blob.SHA256)
			results <- result{blobName: blob.Name, path: path, err: err}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	paths := make(map[string]string)
	var errs []error
	for r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
		} else {
			paths[r.blobName] = r.path
		}
	}

	if len(errs) > 0 {
		return paths, fmt.Errorf("failed to download %d blob(s): %v", len(errs), errs[0])
	}

	return paths, nil
}

// ValidateChecksum computes SHA256 of a file and compares to expected.
// Expected should be a lowercase hex string.
func (fm *FirmwareMgr) ValidateChecksum(filePath, expectedSHA256 string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to read file for checksum: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actual)
	}
	return nil
}

// ComputeChecksum returns the SHA256 hex string of a file.
func (fm *FirmwareMgr) ComputeChecksum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// CleanSocCache removes all cached blobs for a given SoC.
func (fm *FirmwareMgr) CleanSocCache(soc string) error {
	socDir := filepath.Join(fm.cacheDir, soc)
	if err := os.RemoveAll(socDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean SoC cache for %s: %w", soc, err)
	}
	return nil
}

// ListCachedBlobs returns all cached blob names for a given SoC.
func (fm *FirmwareMgr) ListCachedBlobs(soc string) ([]string, error) {
	socDir := filepath.Join(fm.cacheDir, soc)
	entries, err := os.ReadDir(socDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list cached blobs for %s: %w", soc, err)
	}

	var blobs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			blobs = append(blobs, entry.Name())
		}
	}
	return blobs, nil
}

// downloadBlob downloads a file from URL to destPath.
func (fm *FirmwareMgr) downloadBlob(url, destPath string) error {
	resp, err := fm.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP GET failed for %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	// Write to a temporary file first (atomic write)
	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = io.Copy(f, resp.Body)
	_ = f.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write blob data: %w", err)
	}

	// Rename to final destination (atomic on most systems)
	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize blob file: %w", err)
	}

	return nil
}

// --- Known SoC Firmware Blob Registry ---

// BlobSpec describes a downloadable firmware blob.
type BlobSpec struct {
	SoC    string // e.g. "rk3588", "rk3566", "h616"
	Name   string // e.g. "rk3588_ddr.bin"
	URL    string // download URL
	SHA256 string // expected checksum (empty = skip validation)
}

// KnownFirmwareBlobs is the curated registry of SoC firmware blobs.
// URLs point to the rockchip-linux/rkbin GitHub repository.
var KnownFirmwareBlobs = map[string][]BlobSpec{
	"rk3588": {
		{
			SoC:    "rk3588",
			Name:   "rk3588_ddr_lp4_2112MHz.bin",
			URL:    "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk35/rk3588_ddr_lp4_2112MHz_v1.16.bin",
			SHA256: "", // Populated by registry or left for runtime validation
		},
		{
			SoC:    "rk3588",
			Name:   "rk3588_spl_loader.bin",
			URL:    "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk35/rk3588_spl_loader_v1.15.113.bin",
			SHA256: "",
		},
	},
	"rk3566": {
		{
			SoC:  "rk3566",
			Name: "rk3566_ddr_1056MHz.bin",
			URL:  "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk35/rk3566_ddr_1056MHz_v1.21.bin",
		},
		{
			SoC:  "rk3566",
			Name: "rk356x_spl_loader.bin",
			URL:  "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk35/rk356x_spl_loader_v1.18.112.bin",
		},
	},
	"rk3399": {
		{
			SoC:  "rk3399",
			Name: "rk3399_ddr_933MHz.bin",
			URL:  "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk33/rk3399_ddr_933MHz_v1.30.bin",
		},
		{
			SoC:  "rk3399",
			Name: "rk3399_miniloader.bin",
			URL:  "https://github.com/rockchip-linux/rkbin/raw/master/bin/rk33/rk3399_miniloader_v1.30.bin",
		},
	},
}

// GetBlobsForSoC returns all known firmware blobs for a given SoC identifier.
// Returns nil if the SoC is unknown.
func GetBlobsForSoC(soc string) []BlobSpec {
	return KnownFirmwareBlobs[soc]
}

// GetBlobByName looks up a specific blob within a SoC's blob list.
func GetBlobByName(soc, name string) *BlobSpec {
	for _, blob := range KnownFirmwareBlobs[soc] {
		if blob.Name == name {
			b := blob
			return &b
		}
	}
	return nil
}
