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

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push all repositories and sync workspace config",
	Long: `Push all repositories to their remotes and sync workspace configuration.

This command will:
  1. Push all repositories with uncommitted changes
  2. Sync workspace configuration (IDE settings, etc.)
  3. Update the repository inventory (REPOS.md)
  4. Commit and push the metarepo itself`,
	RunE: runPush,
}

var (
	pushDryRun     bool
	pushSkipConfig bool
)

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolVar(&pushDryRun, "dry-run", false, "show what would be pushed without actually pushing")
	pushCmd.Flags().BoolVar(&pushSkipConfig, "skip-config", false, "skip syncing workspace configuration")
}

func runPush(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("Pushing from device: %s (%s)\n\n", deviceName, deviceInfo.Serial)

	// Scan for repositories
	repos, err := git.ScanForRepos(".")
	if err != nil {
		return fmt.Errorf("failed to scan for repositories: %w", err)
	}

	// Push all repos
	fmt.Printf("Found %d repositories\n\n", len(repos))

	pushedCount := 0
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

		if pushDryRun {
			status := "would push"
			if !repo.HasChanges {
				status = "no changes"
			}
			fmt.Printf("  [DRY] %s (%s)\n", repo.Name, status)
			continue
		}

		fmt.Printf("  [PUSH] %s... ", repo.Name)

		if err := git.Push(repo.AbsPath); err != nil {
			fmt.Println("FAILED")
			errorCount++
		} else {
			fmt.Println("OK")
			pushedCount++
		}
	}

	fmt.Println()

	// Sync workspace config
	if !pushSkipConfig && !pushDryRun {
		fmt.Println("Syncing workspace configuration...")
		if err := syncWorkspaceConfig(deviceName); err != nil {
			fmt.Printf("Warning: Failed to sync config: %v\n", err)
		} else {
			fmt.Println("Workspace configuration synced.")
		}
		fmt.Println()
	}

	// Update device last sync time
	if registry != nil && !pushDryRun {
		registry.UpdateLastSync(deviceInfo.Serial)
		registry.Save(devicesPath)
	}

	// Summary
	fmt.Println("Summary:")
	fmt.Printf("  Pushed:  %d\n", pushedCount)
	fmt.Printf("  Skipped: %d\n", skippedCount)
	if errorCount > 0 {
		fmt.Printf("  Errors:  %d\n", errorCount)
	}

	return nil
}

// syncWorkspaceConfig syncs IDE configs to the workspace-config directory
func syncWorkspaceConfig(deviceName string) error {
	configPath := filepath.Join(".metarepo", "config.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	destDir := filepath.Join(".metarepo", "workspace-config", deviceName)

	// Sync each IDE config
	syncPaths := []string{}
	syncPaths = append(syncPaths, cfg.Sync.IDE.Cursor...)
	syncPaths = append(syncPaths, cfg.Sync.IDE.Claude...)
	syncPaths = append(syncPaths, cfg.Sync.IDE.VSCode...)

	for _, srcPath := range syncPaths {
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		destPath := filepath.Join(destDir, srcPath)

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Use rsync for syncing (cross-platform alternative could be implemented)
		cmd := exec.Command("rsync", "-a", "--delete",
			"--exclude", ".git/",
			"--exclude", "node_modules/",
			"--exclude", ".venv/",
			"--exclude", "venv/",
			"--exclude", "__pycache__/",
			"--exclude", ".DS_Store",
			srcPath, destPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to sync %s: %w", srcPath, err)
		}
	}

	return nil
}
