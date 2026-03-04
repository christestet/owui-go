package pipelines

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/cobra"
)

var pipelinesCmd = &cobra.Command{
	Use:   "pipelines",
	Short: "Manage pipelines and valves in an Open WebUI instance",
}

var valvesCmd = &cobra.Command{
	Use:   "valves",
	Short: "Manage pipe valves",
}

// Register adds all pipeline-related subcommands to the root command.
func Register(rootCmd *cobra.Command) {
	pipelinesCmd.AddCommand(listCmd)
	pipelinesCmd.AddCommand(showCmd)
	pipelinesCmd.AddCommand(addCmd)
	pipelinesCmd.AddCommand(uploadCmd)
	pipelinesCmd.AddCommand(removeCmd)

	valvesCmd.AddCommand(valvesShowCmd)
	valvesCmd.AddCommand(valvesSpecCmd)
	valvesCmd.AddCommand(valvesUpdateCmd)
	pipelinesCmd.AddCommand(valvesCmd)

	rootCmd.AddCommand(pipelinesCmd)
}

// resolveClient loads config, resolves the target instance, and returns an API client.
func resolveClient(cmd *cobra.Command) (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	targetInstance, _ := cmd.Flags().GetString("instance")
	if targetInstance == "" {
		targetInstance = cfg.ActiveInstance
	}
	if targetInstance == "" {
		return nil, fmt.Errorf("no active instance configured; use 'owui instances use <name>' or pass --instance")
	}

	inst, ok := cfg.Instances[targetInstance]
	if !ok {
		return nil, fmt.Errorf("instance %q not found in config", targetInstance)
	}

	return api.NewClient(inst.URL, inst.APIKey, cfg.Settings.TimeoutSeconds), nil
}

// resolveClientForCompletion is a best-effort version for shell completion callbacks.
func resolveClientForCompletion() *api.Client {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	inst, ok := cfg.Instances[cfg.ActiveInstance]
	if !ok {
		return nil
	}
	return api.NewClient(inst.URL, inst.APIKey, cfg.Settings.TimeoutSeconds)
}

func pipeCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client := resolveClientForCompletion()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	inv, err := buildInventory(context.Background(), client, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	selected := make(map[string]bool)
	for _, a := range args {
		selected[a] = true
	}

	keys := make([]string, 0, len(inv.PipeIndex))
	for pipeID := range inv.PipeIndex {
		keys = append(keys, pipeID)
	}
	sort.Strings(keys)

	var comps []string
	for _, pipeID := range keys {
		if selected[pipeID] {
			continue
		}
		if !strings.HasPrefix(pipeID, toComplete) {
			continue
		}
		candidates := inv.PipeIndex[pipeID]
		if len(candidates) == 1 {
			c := candidates[0]
			comps = append(comps, fmt.Sprintf("%s\t%s / idx=%d", pipeID, c.RegistrationID, c.URLIdx))
			continue
		}
		idxs := make([]string, 0, len(candidates))
		for _, c := range candidates {
			idxs = append(idxs, fmt.Sprintf("%d", c.URLIdx))
		}
		comps = append(comps, fmt.Sprintf("%s\tambiguous across idx=[%s]", pipeID, strings.Join(idxs, ",")))
	}
	return comps, cobra.ShellCompDirectiveNoFileComp
}

func registrationCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client := resolveClientForCompletion()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	inv, err := buildInventory(context.Background(), client, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var comps []string
	for _, r := range inv.Registrations {
		if strings.HasPrefix(r.RegistrationID, toComplete) {
			comps = append(comps, fmt.Sprintf("%s\tidx=%d %s", r.RegistrationID, r.URLIdx, r.URL))
		}
	}
	return comps, cobra.ShellCompDirectiveNoFileComp
}
