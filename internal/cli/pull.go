package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/JPlanken/metarepo-cli/internal/config"
	"github.com/JPlanken/metarepo-cli/internal/device"
	"github.com/JPlanken/metarepo-cli/internal/git"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull all repositories and sync workspace config",
	Long: `Pull all repositories from their remotes and sync workspace configuration.

This command will:
  1. Pull the metarepo to get latest state
  2. Clone any new repositories found in manifest
  3. Pull all existing repositories
  4. Optionally sync workspace configuration from another device`,
	RunE: runPull,
}

var (
	pullDryRun     bool
	pullSkipConfig bool
	pullFromDevice string
)

func init() {
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().BoolVar(&pullDryRun, "dry-run", false, "show what would be pulled without actually pulling")
	pullCmd.Flags().BoolVar(&pullSkipConfig, "skip-config", false, "skip syncing workspace configuration")
	pullCmd.Flags().StringVar(&pullFromDevice, "from", "", "sync config from specific device")
}

func runPull(cmd *cobra.Command, args []string) error {
	// Get device info
	deviceInfo, err := device.GetCurrentDevice()
	if err != nil {
		return fmt.Errorf("failed to get device info: %w", err)
	}

	// Load device registry to get device name
	devicesPath := filepath.Join(".metarepo", "devices.yaml")
	registry, err := config.LoadDeviceRegistry(devicesPath)
	if err != nil {
		fmt.Println("Warning: Could not load device registry")
	}

	deviceName := deviceInfo.Hostname
	if registry != nil {
		if d := registry.FindDevice(deviceInfo.Serial); d != nil {
			deviceName = d.Name
		}
	}

	fmt.Printf("Pulling to device: %s (%s)\n\n", deviceName, deviceInfo.Serial)

	// Load manifest to check for new repos
	manifestPath := filepath.Join(".metarepo", "manifest.yaml")
	manifest, _ := config.LoadManifest(manifestPath)

	// Clone new repos from manifest
	if manifest != nil && len(manifest.Repositories) > 0 {
		fmt.Println("Checking for new repositories...")
		newCount := 0

		for _, repo := range manifest.Repositories {
			repoPath := repo.Path
			if repoPath == "" {
				repoPath = repo.Name
			}

			if _, err := os.Stat(repoPath); os.IsNotExist(err) {
				if repo.URL == "" {
					continue
				}

				if pullDryRun {
					fmt.Printf("  [DRY] Would clone: %s\n", repo.Name)
					newCount++
					continue
				}

				fmt.Printf("  [CLONE] %s... ", repo.Name)
				if err := git.Clone(repo.URL, repoPath); err != nil {
					fmt.Println("FAILED")
				} else {
					fmt.Println("OK")
					newCount++
				}
			}
		}

		if newCount > 0 {
			fmt.Printf("Cloned %d new repositories\n", newCount)
		} else {
			fmt.Println("No new repositories to clone")
		}
		fmt.Println()
	}

	// Scan for existing repositories
	repos, err := git.ScanForRepos(".")
	if err != nil {
		return fmt.Errorf("failed to scan for repositories: %w", err)
	}

	// Pull all repos
	fmt.Printf("Pulling %d repositories\n\n", len(repos))

	pulledCount := 0
	skippedCount := 0
	errorCount := 0

	for _, repo := range repos {
		// Skip repos without remote
		if !repo.HasRemote {
			fmt.Printf("  [SKIP] %s (no remote)\n", repo.Name)
			skippedCount++
			continue
		}

		// Skip detached HEAD
		if repo.IsDetached {
			fmt.Printf("  [SKIP] %s (detached HEAD)\n", repo.Name)
			skippedCount++
			continue
		}

		if pullDryRun {
			fmt.Printf("  [DRY] %s (would pull)\n", repo.Name)
			continue
		}

		fmt.Printf("  [PULL] %s... ", repo.Name)

		if err := git.Pull(repo.AbsPath); err != nil {
			fmt.Println("FAILED")
			errorCount++
		} else {
			fmt.Println("OK")
			pulledCount++
		}
	}

	fmt.Println()

	// Sync workspace config from another device
	if !pullSkipConfig && !pullDryRun && pullFromDevice != "" {
		fmt.Printf("Syncing workspace configuration from %s...\n", pullFromDevice)
		if err := pullWorkspaceConfig(pullFromDevice, deviceName); err != nil {
			fmt.Printf("Warning: Failed to sync config: %v\n", err)
		} else {
			fmt.Println("Workspace configuration synced.")
		}
		fmt.Println()
	}

	// Update device last sync time
	if registry != nil && !pullDryRun {
		registry.UpdateLastSync(deviceInfo.Serial)
		registry.Save(devicesPath)
	}

	// Summary
	fmt.Println("Summary:")
	fmt.Printf("  Pulled:  %d\n", pulledCount)
	fmt.Printf("  Skipped: %d\n", skippedCount)
	if errorCount > 0 {
		fmt.Printf("  Errors:  %d\n", errorCount)
	}

	return nil
}

// pullWorkspaceConfig syncs IDE configs from another device's workspace-config
func pullWorkspaceConfig(fromDevice, toDevice string) error {
	srcDir := filepath.Join(".metarepo", "workspace-config", fromDevice)
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return fmt.Errorf("no configuration found for device: %s", fromDevice)
	}

	configPath := filepath.Join(".metarepo", "config.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Sync each IDE config back to the workspace root
	syncPaths := []string{}
	syncPaths = append(syncPaths, cfg.Sync.IDE.Cursor...)
	syncPaths = append(syncPaths, cfg.Sync.IDE.Claude...)
	syncPaths = append(syncPaths, cfg.Sync.IDE.VSCode...)

	for _, destPath := range syncPaths {
		srcPath := filepath.Join(srcDir, destPath)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Use rsync for syncing
		cmd := exec.Command("rsync", "-a", "--delete",
			"--exclude", ".git/",
			"--exclude", "node_modules/",
			"--exclude", ".venv/",
			"--exclude", "venv/",
			"--exclude", "__pycache__/",
			"--exclude", ".DS_Store",
			srcPath, destPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to sync %s: %w", destPath, err)
		}
	}

	return nil
}
