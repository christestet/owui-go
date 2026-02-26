package instances

import (
	"fmt"

	"github.com/spf13/cobra"
)

var instancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "Manage Open WebUI instances",
	Long:  `List, use, add, or remove Open WebUI instances.`,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all instances",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Instances List: Not implemented yet")
	},
}

// Register adds the instances subcommands to the root command
func Register(rootCmd *cobra.Command) {
	instancesCmd.AddCommand(listCmd)
	// Add other commands as we build them out
	// instancesCmd.AddCommand(useCmd)
	// instancesCmd.AddCommand(addCmd)
	// instancesCmd.AddCommand(removeCmd)

	rootCmd.AddCommand(instancesCmd)
}
