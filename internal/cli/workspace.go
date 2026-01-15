package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/JPlanken/metarepo-cli/internal/config"
	"github.com/JPlanken/metarepo-cli/internal/device"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Workspace information and management",
	Long:  `Commands for viewing and managing workspace information.`,
}

var workspaceInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show workspace information",
	Long:  `Display detailed information about the current workspace including ID, location, and device.`,
	RunE:  runWorkspaceInfo,
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
	workspaceCmd.AddCommand(workspaceInfoCmd)
}

func runWorkspaceInfo(cmd *cobra.Command, args []string) error {
	// Load workspace config
	configPath := filepath.Join(".metarepo", "config.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("not in a metarepo workspace (no .metarepo/config.yaml found)")
	}

	// Get current device info
	deviceInfo, err := device.GetCurrentDevice()
	if err != nil {
		return fmt.Errorf("failed to get device info: %w", err)
	}

	// Load device registry
	devicesPath := filepath.Join(".metarepo", "devices.yaml")
	registry, _ := config.LoadDeviceRegistry(devicesPath)

	deviceName := deviceInfo.Hostname
	if registry != nil {
		if d := registry.FindDevice(deviceInfo.Serial); d != nil {
			deviceName = d.Name
		}
	}

	// Get current working directory
	cwd, _ := os.Getwd()

	fmt.Println("Workspace Information:")
	fmt.Println()
	fmt.Printf("  ID:       %s\n", cfg.Workspace.ID)
	fmt.Printf("  Name:     %s\n", cfg.Workspace.Name)
	fmt.Println()
	fmt.Println("Location:")
	fmt.Printf("  Device:   %s\n", deviceName)
	fmt.Printf("  Serial:   %s\n", deviceInfo.Serial)
	fmt.Printf("  Path:     %s\n", cwd)
	fmt.Printf("  Platform: %s/%s\n", deviceInfo.Platform, deviceInfo.Arch)
	fmt.Println()

	// Show sync info
	fmt.Println("Sync Configuration:")
	fmt.Printf("  Enabled:  %t\n", cfg.Sync.Enabled)
	if cfg.Sync.Remote != "" {
		fmt.Printf("  Remote:   %s\n", cfg.Sync.Remote)
	}
	fmt.Printf("  IDE sync: cursor=%d, claude=%d, vscode=%d paths\n",
		len(cfg.Sync.IDE.Cursor), len(cfg.Sync.IDE.Claude), len(cfg.Sync.IDE.VSCode))

	// Show registered devices
	if registry != nil && len(registry.Devices) > 0 {
		fmt.Println()
		fmt.Printf("Registered Devices: %d\n", len(registry.Devices))
		for _, d := range registry.Devices {
			current := ""
			if d.Serial == deviceInfo.Serial {
				current = " (current)"
			}
			fmt.Printf("  - %s%s\n", d.Name, current)
		}
	}

	return nil
}
