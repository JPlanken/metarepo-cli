package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/JPlanken/metarepo-cli/internal/config"
	"github.com/JPlanken/metarepo-cli/internal/git"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Repository management commands",
	Long:  `Commands for managing repositories in the metarepo workspace.`,
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repositories",
	Long:  `List all git repositories in the workspace.`,
	RunE:  runRepoList,
}

var repoStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all repositories",
	Long:  `Display the git status of all repositories in the workspace.`,
	RunE:  runRepoStatus,
}

var repoAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a repository to the workspace",
	Long:  `Clone a repository and add it to the workspace manifest.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoAdd,
}

var repoScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for repositories and update manifest",
	Long:  `Scan the workspace for git repositories and update the manifest.`,
	RunE:  runRepoScan,
}

var repoRuntimesCmd = &cobra.Command{
	Use:   "runtimes",
	Short: "Show detected runtimes/tools per repository",
	Long:  `Scan repositories and display detected programming languages and tool versions.`,
	RunE:  runRepoRuntimes,
}

var (
	repoListShort    bool
	repoListRuntimes bool
)

func init() {
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoStatusCmd)
	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoScanCmd)
	repoCmd.AddCommand(repoRuntimesCmd)

	repoListCmd.Flags().BoolVarP(&repoListShort, "short", "s", false, "short output format")
	repoListCmd.Flags().BoolVarP(&repoListRuntimes, "runtimes", "r", false, "show detected runtimes")
}

func runRepoList(cmd *cobra.Command, args []string) error {
	repos, err := git.ScanForRepos(".")
	if err != nil {
		return fmt.Errorf("failed to scan for repositories: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}

	if repoListShort {
		for _, repo := range repos {
			fmt.Println(repo.Name)
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if repoListRuntimes {
		fmt.Fprintln(w, "NAME\tBRANCH\tRUNTIMES\tPATH\t")
	} else {
		fmt.Fprintln(w, "NAME\tBRANCH\tLAST COMMIT\tPATH\t")
	}

	for _, repo := range repos {
		if repoListRuntimes {
			runtimes := git.DetectRuntimes(repo.AbsPath)
			runtimeStr := "-"
			if len(runtimes) > 0 {
				var parts []string
				for _, rt := range runtimes {
					if rt.Version != "" {
						parts = append(parts, fmt.Sprintf("%s:%s", rt.Language, rt.Version))
					} else {
						parts = append(parts, rt.Language)
					}
				}
				runtimeStr = strings.Join(parts, ", ")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n",
				repo.Name,
				repo.Branch,
				runtimeStr,
				repo.Path,
			)
		} else {
			lastCommit := repo.LastCommit.Date.Format("2006-01-02")
			if repo.LastCommit.Date.IsZero() {
				lastCommit = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n",
				repo.Name,
				repo.Branch,
				lastCommit,
				repo.Path,
			)
		}
	}
	w.Flush()

	fmt.Printf("\nTotal: %d repositories\n", len(repos))

	return nil
}

func runRepoRuntimes(cmd *cobra.Command, args []string) error {
	repos, err := git.ScanForRepos(".")
	if err != nil {
		return fmt.Errorf("failed to scan for repositories: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}

	// Collect runtime stats
	langCount := make(map[string]int)

	for _, repo := range repos {
		runtimes := git.DetectRuntimes(repo.AbsPath)
		if len(runtimes) == 0 {
			continue
		}

		fmt.Printf("%s:\n", repo.Name)
		for _, rt := range runtimes {
			langCount[rt.Language]++
			version := rt.Version
			if version == "" {
				version = "(unknown)"
			}
			fmt.Printf("  %s %s\n", rt.Language, version)
			if len(rt.Files) > 0 {
				fmt.Printf("    files: %s\n", strings.Join(rt.Files, ", "))
			}
		}
		fmt.Println()
	}

	// Summary
	if len(langCount) > 0 {
		fmt.Println("Summary:")
		for lang, count := range langCount {
			fmt.Printf("  %s: %d repos\n", lang, count)
		}
	}

	return nil
}

func runRepoStatus(cmd *cobra.Command, args []string) error {
	repos, err := git.ScanForRepos(".")
	if err != nil {
		return fmt.Errorf("failed to scan for repositories: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tBRANCH\tSTATUS\tREMOTE\t")

	cleanCount := 0
	dirtyCount := 0

	for _, repo := range repos {
		status := "clean"
		if repo.HasChanges {
			status = "modified"
			dirtyCount++
		} else {
			cleanCount++
		}

		remote := "yes"
		if !repo.HasRemote {
			remote = "no"
		}

		if repo.IsDetached {
			repo.Branch = "(detached)"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n",
			repo.Name,
			repo.Branch,
			status,
			remote,
		)
	}
	w.Flush()

	fmt.Printf("\nTotal: %d repositories (%d clean, %d modified)\n", len(repos), cleanCount, dirtyCount)

	return nil
}

func runRepoAdd(cmd *cobra.Command, args []string) error {
	url := args[0]

	// Extract repo name from URL
	name := filepath.Base(url)
	name = name[:len(name)-len(filepath.Ext(name))] // Remove .git extension

	// Clone the repository
	fmt.Printf("Cloning %s...\n", url)
	if err := git.Clone(url, name); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Update manifest
	manifestPath := filepath.Join(".metarepo", "manifest.yaml")
	manifest, err := config.LoadManifest(manifestPath)
	if err != nil {
		// Create new manifest if it doesn't exist
		manifest = &config.Manifest{
			Version:      "1.0",
			Repositories: []config.Repository{},
		}
	}

	// Get repo info
	repoInfo, err := git.GetRepoInfo(name)
	if err != nil {
		return fmt.Errorf("failed to get repository info: %w", err)
	}

	// Add to manifest
	manifest.Repositories = append(manifest.Repositories, config.Repository{
		Name:   name,
		Path:   name,
		URL:    repoInfo.URL,
		Branch: repoInfo.Branch,
	})

	if err := manifest.Save(manifestPath); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	fmt.Printf("Repository '%s' added successfully.\n", name)
	return nil
}

func runRepoScan(cmd *cobra.Command, args []string) error {
	fmt.Println("Scanning for repositories...")

	repos, err := git.ScanForRepos(".")
	if err != nil {
		return fmt.Errorf("failed to scan for repositories: %w", err)
	}

	// Load or create manifest
	manifestPath := filepath.Join(".metarepo", "manifest.yaml")
	manifest, err := config.LoadManifest(manifestPath)
	if err != nil {
		manifest = &config.Manifest{
			Version:      "1.0",
			Repositories: []config.Repository{},
		}
	}

	// Update manifest with found repos
	manifest.Repositories = make([]config.Repository, 0, len(repos))
	for _, repo := range repos {
		manifest.Repositories = append(manifest.Repositories, config.Repository{
			Name:   repo.Name,
			Path:   repo.Path,
			URL:    repo.URL,
			Branch: repo.Branch,
		})
	}

	if err := manifest.Save(manifestPath); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	fmt.Printf("Found and registered %d repositories.\n", len(repos))
	return nil
}
