package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

// Version info set from main
var (
	versionStr = "dev"
	commitStr  = "none"
	dateStr    = "unknown"
)

// SetVersionInfo sets version information from build flags
func SetVersionInfo(version, commit, date string) {
	versionStr = version
	commitStr = commit
	dateStr = date
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "metarepo",
	Short: "Multi-repository workspace management tool",
	Long: `metarepo is a CLI tool for managing multi-repository workspaces
with cross-device synchronization.

It enables you to:
  - Sync multiple git repositories across devices
  - Manage IDE configurations (.cursor, .claude, .vscode)
  - Generate repository inventories
  - Execute commands across all repositories`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/metarepo/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error finding home directory:", err)
			os.Exit(1)
		}

		// Search for config in these locations (in order of priority)
		viper.AddConfigPath(".")                                      // Current directory
		viper.AddConfigPath(filepath.Join(home, ".config", "metarepo")) // User config
		viper.AddConfigPath("/etc/metarepo")                          // System config
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Read environment variables with METAREPO_ prefix
	viper.SetEnvPrefix("METAREPO")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
