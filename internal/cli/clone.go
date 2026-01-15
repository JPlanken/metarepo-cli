package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/JPlanken/metarepo-cli/internal/config"
	"github.com/JPlanken/metarepo-cli/internal/git"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone all repositories from the manifest",
	Long: `Clone all repositories defined in the workspace manifest.

This is typically used when setting up a new device. It will:
  1. Read the manifest file
  2. Clone all repositories that don't exist locally`,
	RunE: runClone,
}

var (
	cloneDryRun   bool
	cloneParallel int
)

func init() {
	rootCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().BoolVar(&cloneDryRun, "dry-run", false, "show what would be cloned without actually cloning")
	cloneCmd.Flags().IntVarP(&cloneParallel, "parallel", "p", 1, "number of parallel clones (default 1)")
}

func runClone(cmd *cobra.Command, args []string) error {
	// Load manifest
	manifestPath := filepath.Join(".metarepo", "manifest.yaml")
	manifest, err := config.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if len(manifest.Repositories) == 0 {
		fmt.Println("No repositories in manifest.")
		return nil
	}

	fmt.Printf("Found %d repositories in manifest\n\n", len(manifest.Repositories))

	clonedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, repo := range manifest.Repositories {
		repoPath := repo.Path
		if repoPath == "" {
			repoPath = repo.Name
		}

		// Check if already exists
		if _, err := os.Stat(repoPath); err == nil {
			if git.IsGitRepo(repoPath) {
				fmt.Printf("  [SKIP] %s (already exists)\n", repo.Name)
				skippedCount++
				continue
			}
		}

		// Check if URL is available
		if repo.URL == "" {
			fmt.Printf("  [SKIP] %s (no URL)\n", repo.Name)
			skippedCount++
			continue
		}

		if cloneDryRun {
			fmt.Printf("  [DRY] %s → %s\n", repo.Name, repoPath)
			continue
		}

		fmt.Printf("  [CLONE] %s... ", repo.Name)

		if err := git.Clone(repo.URL, repoPath); err != nil {
			fmt.Println("FAILED")
			errorCount++
		} else {
			fmt.Println("OK")
			clonedCount++
		}
	}

	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  Cloned:  %d\n", clonedCount)
	fmt.Printf("  Skipped: %d\n", skippedCount)
	if errorCount > 0 {
		fmt.Printf("  Errors:  %d\n", errorCount)
	}

	return nil
}
