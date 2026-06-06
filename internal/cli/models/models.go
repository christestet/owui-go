package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Manage models in an Open WebUI instance",
}

// Register adds all model-related subcommands to the root command.
func Register(rootCmd *cobra.Command) {
	modelsCmd.AddCommand(listCmd)
	modelsCmd.AddCommand(showCmd)
	modelsCmd.AddCommand(setStatusCmd)
	modelsCmd.AddCommand(setVisibilityCmd)
	modelsCmd.AddCommand(addToGroupCmd)
	modelsCmd.AddCommand(removeFromGroupCmd)

	rootCmd.AddCommand(modelsCmd)
}

// findModelByID looks up a model by ID from a model list.
func findModelByID(models []api.ModelAccessResponse, id string) (*api.ModelAccessResponse, error) {
	for i := range models {
		if models[i].ID == id {
			return &models[i], nil
		}
	}
	return nil, fmt.Errorf("model %q not found", id)
}

// findModelByNameOrID looks up a model by name or ID from a model list.
func findModelByNameOrID(models []api.ModelAccessResponse, nameOrID string) (*api.ModelAccessResponse, error) {
	for i := range models {
		if models[i].ID == nameOrID || models[i].Name == nameOrID {
			return &models[i], nil
		}
	}
	return nil, fmt.Errorf("model %q not found", nameOrID)
}

// isPublic returns true if the model has no access grants (public).
func isPublic(m api.ModelAccessResponse) bool {
	return len(m.AccessGrants) == 0
}

// isPrivate returns true if the model has access grants (private).
func isPrivate(m api.ModelAccessResponse) bool {
	return len(m.AccessGrants) > 0
}

// modelVisibility returns "public" or "private" based on access grants.
func modelVisibility(m api.ModelAccessResponse) string {
	if isPublic(m) {
		return "public"
	}
	return "private"
}

// modelStatus returns "enabled" or "disabled" based on is_active.
func modelStatus(m api.ModelAccessResponse) string {
	if m.IsActive {
		return "enabled"
	}
	return "disabled"
}

// modelCompletionFunc completes model IDs showing "Name (id)" format.
func modelCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client := shared.ResolveClientForCompletion(cmd)
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	models, err := client.ListModels(context.Background())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var comps []string
	for _, m := range models {
		if strings.HasPrefix(m.ID, toComplete) || strings.HasPrefix(m.Name, toComplete) {
			comps = append(comps, fmt.Sprintf("%s\t%s", m.ID, m.Name))
		}
	}
	return comps, cobra.ShellCompDirectiveNoFileComp
}

// smartModelCompletionFunc returns a completion function that filters models based on their state.
// filterFn determines which models to include in completions.
func smartModelCompletionFunc(filterFn func(api.ModelAccessResponse) bool) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client := shared.ResolveClientForCompletion(cmd)
		if client == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		models, err := client.ListModels(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		selected := make(map[string]bool)
		for _, a := range args {
			selected[a] = true
		}
		var comps []string
		for _, m := range models {
			if selected[m.ID] {
				continue
			}
			if !filterFn(m) {
				continue
			}
			if strings.HasPrefix(m.ID, toComplete) || strings.HasPrefix(m.Name, toComplete) {
				comps = append(comps, fmt.Sprintf("%s\t%s", m.ID, m.Name))
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	}
}

// multiModelCompletionFunc completes multiple model IDs, excluding already-selected ones.
func multiModelCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := shared.ResolveClientForCompletion(cmd)
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	models, err := client.ListModels(context.Background())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	selected := make(map[string]bool)
	for _, a := range args {
		selected[a] = true
	}
	var comps []string
	for _, m := range models {
		if selected[m.ID] {
			continue
		}
		if strings.HasPrefix(m.ID, toComplete) || strings.HasPrefix(m.Name, toComplete) {
			comps = append(comps, fmt.Sprintf("%s\t%s", m.ID, m.Name))
		}
	}
	return comps, cobra.ShellCompDirectiveNoFileComp
}

// localGroupCompletionFunc completes only local group names.
func localGroupCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := shared.ResolveClientForCompletion(cmd)
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	groups, err := client.ListGroups(context.Background())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var comps []string
	for _, g := range shared.FilterLocalGroups(groups) {
		if strings.HasPrefix(g.Name, toComplete) {
			comps = append(comps, g.Name)
		}
	}
	return comps, cobra.ShellCompDirectiveNoFileComp
}
