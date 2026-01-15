package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Config represents the main configuration
type Config struct {
	Version    string          `yaml:"version"`
	Workspace  WorkspaceConfig `yaml:"workspace"`
	Repos      ReposConfig     `yaml:"repos"`
	Sync       SyncConfig      `yaml:"sync"`
	Inventory  InventoryConfig `yaml:"inventory"`
	Logging    LoggingConfig   `yaml:"logging"`
}

// ReposConfig holds repository filtering settings
type ReposConfig struct {
	Exclude []string `yaml:"exclude,omitempty"` // Repo names or patterns to exclude (e.g., "temp-*", "test-repo")
}

// WorkspaceConfig holds workspace settings
type WorkspaceConfig struct {
	ID          string `yaml:"id"`                    // Unique workspace UUID for sync collision prevention
	Name        string `yaml:"name"`
	Root        string `yaml:"root"`
	Description string `yaml:"description,omitempty"`
}

// SyncConfig holds synchronization settings
type SyncConfig struct {
	Enabled  bool           `yaml:"enabled"`
	Remote   string         `yaml:"remote"`
	Branch   string         `yaml:"branch,omitempty"`
	IDE      IDEConfig      `yaml:"ide"`
	Conflict ConflictConfig `yaml:"conflict,omitempty"`
}

// IDEConfig holds IDE-specific sync paths
type IDEConfig struct {
	Cursor []string `yaml:"cursor,omitempty"`
	Claude []string `yaml:"claude,omitempty"`
	VSCode []string `yaml:"vscode,omitempty"`
}

// ConflictConfig holds conflict resolution settings
type ConflictConfig struct {
	Strategy string `yaml:"strategy,omitempty"` // newest, local, remote, manual
}

// InventoryConfig holds inventory generation settings
type InventoryConfig struct {
	Output  string   `yaml:"output"`
	Include []string `yaml:"include,omitempty"`
	GroupBy string   `yaml:"group_by,omitempty"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level string `yaml:"level,omitempty"`
	File  string `yaml:"file,omitempty"`
}

// Manifest represents the repository manifest
type Manifest struct {
	Version      string       `yaml:"version"`
	Generated    time.Time    `yaml:"generated"`
	Repositories []Repository `yaml:"repositories"`
}

// Repository represents a single repository in the manifest
type Repository struct {
	Name        string   `yaml:"name"`
	Path        string   `yaml:"path"`
	URL         string   `yaml:"url"`
	Branch      string   `yaml:"branch"`
	Tags        []string `yaml:"tags,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

// DeviceRegistry holds information about known devices
type DeviceRegistry struct {
	Version string   `yaml:"version"`
	Devices []Device `yaml:"devices"`
}

// Device represents a single registered device
type Device struct {
	Serial     string    `yaml:"serial"`
	Name       string    `yaml:"name"`
	Platform   string    `yaml:"platform"`
	Hostname   string    `yaml:"hostname,omitempty"`
	Registered time.Time `yaml:"registered"`
	LastSync   time.Time `yaml:"last_sync,omitempty"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Workspace: WorkspaceConfig{
			ID:   uuid.New().String(),
			Name: "workspace",
			Root: ".",
		},
		Sync: SyncConfig{
			Enabled: true,
			Branch:  "main",
			IDE: IDEConfig{
				Cursor: []string{".cursor/"},
				Claude: []string{".claude/"},
				VSCode: []string{".vscode/"},
			},
			Conflict: ConflictConfig{
				Strategy: "newest",
			},
		},
		Inventory: InventoryConfig{
			Output:  "REPOS.md",
			Include: []string{"name", "url", "branch", "last_commit"},
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves configuration to a file
func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadManifest loads the repository manifest
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return &m, nil
}

// Save saves the manifest to a file
func (m *Manifest) Save(path string) error {
	m.Generated = time.Now()

	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadDeviceRegistry loads the device registry
func LoadDeviceRegistry(path string) (*DeviceRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &DeviceRegistry{Version: "1.0"}, nil
		}
		return nil, err
	}

	var reg DeviceRegistry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, err
	}

	return &reg, nil
}

// Save saves the device registry to a file
func (r *DeviceRegistry) Save(path string) error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// FindDevice finds a device by serial number
func (r *DeviceRegistry) FindDevice(serial string) *Device {
	for i := range r.Devices {
		if r.Devices[i].Serial == serial {
			return &r.Devices[i]
		}
	}
	return nil
}

// AddDevice adds a new device to the registry
func (r *DeviceRegistry) AddDevice(d Device) {
	r.Devices = append(r.Devices, d)
}

// UpdateLastSync updates the last sync time for a device
func (r *DeviceRegistry) UpdateLastSync(serial string) {
	if d := r.FindDevice(serial); d != nil {
		d.LastSync = time.Now()
	}
}

// IsExcluded checks if a repo name matches any exclude pattern
func (c *Config) IsExcluded(repoName string) bool {
	for _, pattern := range c.Repos.Exclude {
		if matched, _ := filepath.Match(pattern, repoName); matched {
			return true
		}
		// Also check exact match
		if pattern == repoName {
			return true
		}
	}
	return false
}
