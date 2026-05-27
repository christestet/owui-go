package tools

import (
	"github.com/christestet/owui-go/internal/api"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage tools in an Open WebUI instance",
}

// Register adds all tool-related subcommands to the root command.
func Register(rootCmd *cobra.Command) {
	toolsCmd.AddCommand(listCmd)
	rootCmd.AddCommand(toolsCmd)
}

// toolIsPublic returns true if the tool has no access grants (visible to all).
func toolIsPublic(t api.Tool) bool {
	return len(t.AccessGrants) == 0
}

// toolVisibility mirrors models' wording: "public" / "private".
func toolVisibility(t api.Tool) string {
	if toolIsPublic(t) {
		return "public"
	}
	return "private"
}
