package instances

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/cobra"
)

var instancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "Manage Open WebUI instances",
	Long:  `List, use, add, or remove Open WebUI instances.`,
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all instances",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		type SafeInstance struct {
			Name    string `json:"name"`
			URL     string `json:"url"`
			APIKey  string `json:"api_key"`
			AddedAt string `json:"added_at"`
			Active  bool   `json:"active"`
		}

		var safes []SafeInstance
		for name, inst := range cfg.Instances {
			safes = append(safes, SafeInstance{
				Name:    name,
				URL:     inst.URL,
				APIKey:  inst.RedactedAPIKey(),
				AddedAt: inst.AddedAt,
				Active:  name == cfg.ActiveInstance,
			})
		}

		if jsonOutput || outputFormat == "json" {
			b, err := json.MarshalIndent(safes, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		if len(safes) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No instances configured.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tURL\tAPI KEY\tACTIVE")
		for _, s := range safes {
			activeMark := ""
			if s.Active {
				activeMark = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.URL, s.APIKey, activeMark)
		}
		w.Flush()

		return nil
	},
}

var useCmd = &cobra.Command{
	Use:   "use <instance-name>",
	Short: "Switch the active instance",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		cfg, err := config.Load()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var comps []string
		for name := range cfg.Instances {
			if strings.HasPrefix(name, toComplete) {
				comps = append(comps, name)
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		instanceName := args[0]
		if _, ok := cfg.Instances[instanceName]; !ok {
			return fmt.Errorf("instance %q not found in config", instanceName)
		}

		cfg.ActiveInstance = instanceName
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Switched to instance %q\n", instanceName)
		return nil
	},
}

// removeCmd represents the remove command we need tab auto complete for the instance name
var removeCmd = &cobra.Command{
	Use:     "remove",
	Short:   "Remove an instance",
	Aliases: []string{"rm"},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		cfg, err := config.Load()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var comps []string
		for name := range cfg.Instances {
			if strings.HasPrefix(name, toComplete) {
				comps = append(comps, name)
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(args) == 0 {
			return fmt.Errorf("instance name is required")
		}

		instanceName := args[0]
		if _, ok := cfg.Instances[instanceName]; !ok {
			return fmt.Errorf("instance %q not found in config", instanceName)
		}

		delete(cfg.Instances, instanceName)

		if cfg.ActiveInstance == instanceName {
			cfg.ActiveInstance = ""
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Instance %q removed\n", instanceName)
		return nil
	},
}

// Register adds the instances subcommands to the root command
func Register(rootCmd *cobra.Command) {
	instancesCmd.AddCommand(listCmd)
	instancesCmd.AddCommand(healthCmd)
	instancesCmd.AddCommand(useCmd)
	instancesCmd.AddCommand(addCmd)
	instancesCmd.AddCommand(removeCmd)

	rootCmd.AddCommand(instancesCmd)
}
