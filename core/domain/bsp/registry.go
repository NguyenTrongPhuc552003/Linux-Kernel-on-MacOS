// Package bsp provides board support package management for ELMOS.
// This file implements the BSP registry client for fetching machine definitions.
package bsp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NguyenTrongPhuc552003/elmos/core/config"
	"gopkg.in/yaml.v3"
)

// defaultCacheTTL is the time after which cached registry data is considered stale.
const defaultCacheTTL = 24 * time.Hour

// RegistryClient fetches machine definitions from a BSP registry.
// It is offline-first: always checks local cache before making network calls.
type RegistryClient struct {
	name       string
	baseURL    string
	cacheDir   string
	cacheTTL   time.Duration
	httpClient *http.Client
}

// RegistryIndex is the machine listing returned by the registry index endpoint.
type RegistryIndex struct {
	Version  string         `json:"version" yaml:"version"`
	Machines []MachineEntry `json:"machines" yaml:"machines"`
	Updated  time.Time      `json:"updated" yaml:"updated"`
}

// MachineEntry is a single entry in the registry index.
type MachineEntry struct {
	Name         string   `json:"name" yaml:"name"`
	Manufacturer string   `json:"manufacturer" yaml:"manufacturer"`
	Description  string   `json:"description" yaml:"description"`
	Tags         []string `json:"tags" yaml:"tags"`
	Path         string   `json:"path" yaml:"path"` // Relative to registry base URL
}

// NewRegistryClient creates a new BSP registry client.
func NewRegistryClient(name, baseURL, cacheDir string) *RegistryClient {
	return &RegistryClient{
		name:     name,
		baseURL:  strings.TrimRight(baseURL, "/"),
		cacheDir: filepath.Join(cacheDir, "registry", name),
		cacheTTL: defaultCacheTTL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchMachine retrieves a machine definition by name.
// Checks local cache first; fetches from registry if missing or expired.
func (rc *RegistryClient) FetchMachine(machineName string) (*config.MachineDefinition, error) {
	// Check cache first
	if def, err := rc.loadFromCache(machineName); err == nil {
		return def, nil
	}

	// Fetch from registry
	def, err := rc.fetchMachineFromNetwork(machineName)
	if err != nil {
		return nil, err
	}

	// Save to cache for offline use
	if cacheErr := rc.saveToCache(machineName, def); cacheErr != nil {
		log.Printf("WARNING: bsp registry cache write failed for machine %q: %v", machineName, cacheErr)
	}

	return def, nil
}

// ListMachines returns all available machines from this registry.
// Uses cached index if available and fresh; fetches if not.
func (rc *RegistryClient) ListMachines() ([]MachineEntry, error) {
	// Check for cached index
	indexPath := filepath.Join(rc.cacheDir, "index.json")
	if info, err := os.Stat(indexPath); err == nil {
		if time.Since(info.ModTime()) < rc.cacheTTL {
			return rc.loadIndexFromCache(indexPath)
		}
	}

	// Fetch fresh index
	index, err := rc.fetchIndex()
	if err != nil {
		// Fallback to stale cache on network error
		if entries, cacheErr := rc.loadIndexFromCache(indexPath); cacheErr == nil {
			return entries, nil
		}
		return nil, fmt.Errorf("failed to fetch registry index for %s: %w", rc.name, err)
	}

	// Save updated index
	if err := rc.saveIndex(indexPath, index); err != nil {
		log.Printf("WARNING: bsp registry index cache write failed for %q: %v", rc.name, err)
	}

	return index.Machines, nil
}

// RefreshCache forcefully re-fetches and updates the local cache.
func (rc *RegistryClient) RefreshCache(machineName string) (*config.MachineDefinition, error) {
	def, err := rc.fetchMachineFromNetwork(machineName)
	if err != nil {
		return nil, err
	}
	if err := rc.saveToCache(machineName, def); err != nil {
		return def, fmt.Errorf("refreshed but failed to update cache: %w", err)
	}
	return def, nil
}

// IsCached returns true if a fresh cache entry exists for the machine.
func (rc *RegistryClient) IsCached(machineName string) bool {
	cachePath := rc.machineCachePath(machineName)
	info, err := os.Stat(cachePath)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < rc.cacheTTL
}

// fetchMachineFromNetwork retrieves the machine YAML from the registry URL.
func (rc *RegistryClient) fetchMachineFromNetwork(machineName string) (*config.MachineDefinition, error) {
	url := fmt.Sprintf("%s/machines/%s.yml", rc.baseURL, machineName)

	resp, err := rc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("network request to %s failed: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("machine '%s' not found in registry '%s'", machineName, rc.name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry '%s' returned HTTP %d for machine '%s'", rc.name, resp.StatusCode, machineName)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var def config.MachineDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse machine definition YAML for '%s': %w", machineName, err)
	}

	if def.Name == "" {
		def.Name = machineName
	}

	return &def, nil
}

// fetchIndex retrieves the registry machine listing.
func (rc *RegistryClient) fetchIndex() (*RegistryIndex, error) {
	url := fmt.Sprintf("%s/index.json", rc.baseURL)

	resp, err := rc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index from %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry index returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read index response: %w", err)
	}

	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse registry index: %w", err)
	}

	return &index, nil
}

// loadFromCache loads a machine definition from the local cache.
// Returns error if cache entry is missing or expired.
func (rc *RegistryClient) loadFromCache(machineName string) (*config.MachineDefinition, error) {
	cachePath := rc.machineCachePath(machineName)

	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, fmt.Errorf("cache miss for machine '%s'", machineName)
	}

	// Check TTL freshness
	if time.Since(info.ModTime()) > rc.cacheTTL {
		return nil, fmt.Errorf("cache entry for '%s' is stale", machineName)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var def config.MachineDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse cached machine definition: %w", err)
	}

	return &def, nil
}

// saveToCache writes a machine definition to local cache as YAML.
func (rc *RegistryClient) saveToCache(machineName string, def *config.MachineDefinition) error {
	if err := os.MkdirAll(rc.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}

	data, err := yaml.Marshal(def)
	if err != nil {
		return fmt.Errorf("failed to marshal machine definition: %w", err)
	}

	cachePath := rc.machineCachePath(machineName)
	return os.WriteFile(cachePath, data, 0644)
}

// loadIndexFromCache reads the cached registry index.
func (rc *RegistryClient) loadIndexFromCache(indexPath string) ([]MachineEntry, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached index: %w", err)
	}

	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse cached index: %w", err)
	}

	return index.Machines, nil
}

// saveIndex writes the registry index to local cache.
func (rc *RegistryClient) saveIndex(indexPath string, index *RegistryIndex) error {
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexPath, data, 0644)
}

// machineCachePath returns the local cache file path for a machine.
func (rc *RegistryClient) machineCachePath(machineName string) string {
	return filepath.Join(rc.cacheDir, machineName+".yml")
}

// --- Multi-Registry Manager ---

// RegistryManager orchestrates multiple BSP registries.
type RegistryManager struct {
	clients  []*RegistryClient
	cacheDir string
}

// NewRegistryManager creates a new manager for multiple registries.
func NewRegistryManager(cacheDir string) *RegistryManager {
	return &RegistryManager{
		cacheDir: cacheDir,
	}
}

// AddRegistry registers a new BSP registry.
func (rm *RegistryManager) AddRegistry(name, baseURL string) {
	client := NewRegistryClient(name, baseURL, rm.cacheDir)
	rm.clients = append(rm.clients, client)
}

// FindMachine searches all registries for a machine definition.
// First match wins (registries are searched in order of addition).
func (rm *RegistryManager) FindMachine(machineName string) (*config.MachineDefinition, string, error) {
	var lastErr error

	for _, client := range rm.clients {
		def, err := client.FetchMachine(machineName)
		if err == nil {
			return def, client.name, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("machine '%s' not found in any registry: %w", machineName, lastErr)
	}
	return nil, "", fmt.Errorf("no registries configured")
}

// ListAllMachines aggregates machine listings from all registries.
// Deduplicates by machine name (first occurrence wins).
func (rm *RegistryManager) ListAllMachines() ([]MachineEntry, error) {
	seen := make(map[string]bool)
	var all []MachineEntry

	for _, client := range rm.clients {
		entries, err := client.ListMachines()
		if err != nil {
			continue // Skip unavailable registries
		}
		for _, entry := range entries {
			if !seen[entry.Name] {
				seen[entry.Name] = true
				all = append(all, entry)
			}
		}
	}

	return all, nil
}

// RegistryCount returns the number of configured registries.
func (rm *RegistryManager) RegistryCount() int {
	return len(rm.clients)
}
