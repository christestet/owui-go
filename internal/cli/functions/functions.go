package functions

import (
	"github.com/christestet/owui-go/internal/api"
	"github.com/spf13/cobra"
)

var functionsCmd = &cobra.Command{
	Use:   "functions",
	Short: "Manage functions in an Open WebUI instance",
}

// Register adds all function-related subcommands to the root command.
func Register(rootCmd *cobra.Command) {
	functionsCmd.AddCommand(listCmd)
	rootCmd.AddCommand(functionsCmd)
}

// functionStatus mirrors models' wording: "enabled" / "disabled".
func functionStatus(f api.Function) string {
	if f.IsActive {
		return "enabled"
	}
	return "disabled"
}

// functionScope returns "global" or "private" when the function is enabled,
// "-" otherwise (scope is only meaningful for active functions).
// Functions in Open WebUI have no per-group access control: they are either
// global (visible to all users) or owner-private.
func functionScope(f api.Function) string {
	if !f.IsActive {
		return "-"
	}
	if f.IsGlobal {
		return "global"
	}
	return "private"
}
