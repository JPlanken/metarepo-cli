package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/JPlanken/metarepo-cli/internal/config"
	"github.com/JPlanken/metarepo-cli/internal/device"
	"github.com/spf13/cobra"
)

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Device management commands",
	Long:  `Commands for managing devices in the metarepo workspace.`,
}

var deviceInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show current device information",
	Long:  `Display information about the current device including serial number, platform, and hostname.`,
	RunE:  runDeviceInfo,
}

var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered devices",
	Long:  `Display all devices registered in the current workspace.`,
	RunE:  runDeviceList,
}

var deviceRegisterCmd = &cobra.Command{
	Use:   "register [name]",
	Short: "Register the current device",
	Long:  `Register the current device in the workspace's device registry.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDeviceRegister,
}

func init() {
	rootCmd.AddCommand(deviceCmd)
	deviceCmd.AddCommand(deviceInfoCmd)
	deviceCmd.AddCommand(deviceListCmd)
	deviceCmd.AddCommand(deviceRegisterCmd)
}

func runDeviceInfo(cmd *cobra.Command, args []string) error {
	info, err := device.GetCurrentDevice()
	if err != nil {
		return fmt.Errorf("failed to get device info: %w", err)
	}

	fmt.Println("Current Device:")
	fmt.Printf("  Serial:   %s\n", info.Serial)
	fmt.Printf("  Platform: %s\n", info.Platform)
	fmt.Printf("  Arch:     %s\n", info.Arch)
	fmt.Printf("  Hostname: %s\n", info.Hostname)
	fmt.Printf("  Username: %s\n", info.Username)

	// Check if registered in current workspace
	devicesPath := filepath.Join(".metarepo", "devices.yaml")
	if registry, err := config.LoadDeviceRegistry(devicesPath); err == nil {
		if d := registry.FindDevice(info.Serial); d != nil {
			fmt.Println()
			fmt.Printf("  Registered as: %s\n", d.Name)
			fmt.Printf("  Registered:    %s\n", d.Registered.Format("2006-01-02 15:04:05"))
			if !d.LastSync.IsZero() {
				fmt.Printf("  Last sync:     %s\n", d.LastSync.Format("2006-01-02 15:04:05"))
			}
		} else {
			fmt.Println()
			fmt.Println("  Status: Not registered in this workspace")
			fmt.Println("  Run 'metarepo device register' to register")
		}
	}

	return nil
}

func runDeviceList(cmd *cobra.Command, args []string) error {
	devicesPath := filepath.Join(".metarepo", "devices.yaml")
	registry, err := config.LoadDeviceRegistry(devicesPath)
	if err != nil {
		return fmt.Errorf("failed to load device registry: %w", err)
	}

	if len(registry.Devices) == 0 {
		fmt.Println("No devices registered.")
		return nil
	}

	// Get current device to mark it
	currentSerial := ""
	if info, err := device.GetCurrentDevice(); err == nil {
		currentSerial = info.Serial
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSERIAL\tPLATFORM\tLAST SYNC\t")

	for _, d := range registry.Devices {
		current := ""
		if d.Serial == currentSerial {
			current = " *"
		}

		lastSync := "never"
		if !d.LastSync.IsZero() {
			lastSync = d.LastSync.Format("2006-01-02 15:04")
		}

		fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t\n", d.Name, current, d.Serial, d.Platform, lastSync)
	}
	w.Flush()

	fmt.Println()
	fmt.Println("* = current device")

	return nil
}

func runDeviceRegister(cmd *cobra.Command, args []string) error {
	info, err := device.GetCurrentDevice()
	if err != nil {
		return fmt.Errorf("failed to get device info: %w", err)
	}

	devicesPath := filepath.Join(".metarepo", "devices.yaml")
	registry, err := config.LoadDeviceRegistry(devicesPath)
	if err != nil {
		return fmt.Errorf("failed to load device registry: %w", err)
	}

	// Check if already registered
	if d := registry.FindDevice(info.Serial); d != nil {
		return fmt.Errorf("device already registered as '%s'", d.Name)
	}

	// Get device name
	deviceName := info.Hostname
	if len(args) > 0 {
		deviceName = args[0]
	}

	// Add device
	registry.AddDevice(info.ToConfigDevice(deviceName))

	if err := registry.Save(devicesPath); err != nil {
		return fmt.Errorf("failed to save device registry: %w", err)
	}

	// Create workspace-config directory for this device
	workspaceConfigDir := filepath.Join(".metarepo", "workspace-config", deviceName)
	if err := os.MkdirAll(workspaceConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace-config directory: %w", err)
	}

	fmt.Printf("Device registered successfully as '%s'\n", deviceName)
	fmt.Printf("  Serial: %s\n", info.Serial)

	return nil
}
