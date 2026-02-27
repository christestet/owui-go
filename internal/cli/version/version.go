package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// These variables will be overridden by GoReleaser during build
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of owui",
	Long:  `All software has versions. This is owui's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "owui version %s (commit: %s, built at: %s)\n", Version, Commit, Date)
	},
}
