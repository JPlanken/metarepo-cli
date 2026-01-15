package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JPlanken/metarepo-cli/internal/config"
	"github.com/JPlanken/metarepo-cli/internal/device"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a new metarepo workspace",
	Long: `Initialize a new metarepo workspace in the current directory or specified path.

This will create:
  - .metarepo/ directory with configuration files
  - config.yaml with workspace settings
  - manifest.yaml for repository tracking
  - devices.yaml for device registry`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

var (
	initName   string
	initForce  bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initName, "name", "n", "", "workspace name")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing configuration")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine workspace root
	workspaceRoot := "."
	if len(args) > 0 {
		workspaceRoot = args[0]
	}

	// Make absolute path
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	metarepoDir := filepath.Join(absRoot, ".metarepo")
	configPath := filepath.Join(metarepoDir, "config.yaml")

	// Check if already initialized
	if _, err := os.Stat(configPath); err == nil && !initForce {
		return fmt.Errorf("workspace already initialized at %s (use --force to overwrite)", absRoot)
	}

	// Get workspace name
	workspaceName := initName
	if workspaceName == "" {
		workspaceName = filepath.Base(absRoot)
		fmt.Printf("Workspace name [%s]: ", workspaceName)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			workspaceName = input
		}
	}

	// Get current device info
	deviceInfo, err := device.GetCurrentDevice()
	if err != nil {
		return fmt.Errorf("failed to get device info: %w", err)
	}

	// Ask for device name
	fmt.Printf("Device name [%s]: ", deviceInfo.Hostname)
	reader := bufio.NewReader(os.Stdin)
	deviceName, _ := reader.ReadString('\n')
	deviceName = strings.TrimSpace(deviceName)
	if deviceName == "" {
		deviceName = deviceInfo.Hostname
	}

	// Create .metarepo directory
	if err := os.MkdirAll(metarepoDir, 0755); err != nil {
		return fmt.Errorf("failed to create .metarepo directory: %w", err)
	}

	// Create default config
	cfg := config.DefaultConfig()
	cfg.Workspace.Name = workspaceName
	cfg.Workspace.Root = absRoot

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create empty manifest
	manifest := &config.Manifest{
		Version:      "1.0",
		Repositories: []config.Repository{},
	}
	manifestPath := filepath.Join(metarepoDir, "manifest.yaml")
	if err := manifest.Save(manifestPath); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	// Create device registry and register this device
	registry := &config.DeviceRegistry{
		Version: "1.0",
	}
	registry.AddDevice(deviceInfo.ToConfigDevice(deviceName))

	devicesPath := filepath.Join(metarepoDir, "devices.yaml")
	if err := registry.Save(devicesPath); err != nil {
		return fmt.Errorf("failed to save device registry: %w", err)
	}

	// Create workspace-config directory for per-device configs
	workspaceConfigDir := filepath.Join(metarepoDir, "workspace-config", deviceName)
	if err := os.MkdirAll(workspaceConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace-config directory: %w", err)
	}

	fmt.Println()
	fmt.Println("Workspace initialized successfully!")
	fmt.Println()
	fmt.Printf("  ID:       %s\n", cfg.Workspace.ID)
	fmt.Printf("  Location: %s\n", absRoot)
	fmt.Printf("  Name:     %s\n", workspaceName)
	fmt.Printf("  Device:   %s (%s)\n", deviceName, deviceInfo.Serial)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Add repositories: metarepo repo add <url>")
	fmt.Println("  2. Configure sync:   metarepo config edit")
	fmt.Println("  3. Push changes:     metarepo push")

	return nil
}
