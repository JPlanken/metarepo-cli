package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Print the version, commit hash, and build date of metarepo.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("metarepo %s\n", versionStr)
		fmt.Printf("  commit: %s\n", commitStr)
		fmt.Printf("  built:  %s\n", dateStr)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
