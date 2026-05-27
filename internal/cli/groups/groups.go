package groups

import (
	"context"
	"strings"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage groups in an Open WebUI instance",
}

// Register adds all group-related subcommands to the root command.
func Register(rootCmd *cobra.Command) {
	groupsCmd.AddCommand(listCmd)
	groupsCmd.AddCommand(addCmd)
	groupsCmd.AddCommand(removeCmd)
	groupsCmd.AddCommand(updateCmd)
	groupsCmd.AddCommand(membersCmd)
	groupsCmd.AddCommand(showModelsCmd)
	groupsCmd.AddCommand(showToolsCmd)
	groupsCmd.AddCommand(addUsersCmd)
	groupsCmd.AddCommand(removeUsersCmd)

	rootCmd.AddCommand(groupsCmd)
}

// groupType returns "oauth" or "local" for a group.
func groupType(g api.Group) string {
	if shared.IsOAuthGroup(g) {
		return "oauth"
	}
	return "local"
}

// filterOAuthGroups returns only OAuth groups.
func filterOAuthGroups(groups []api.Group) []api.Group {
	var oauth []api.Group
	for _, g := range groups {
		if shared.IsOAuthGroup(g) {
			oauth = append(oauth, g)
		}
	}
	return oauth
}

// localGroupCompletionFunc completes only local group names.
func localGroupCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := shared.ResolveClientForCompletion()
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

// multiLocalGroupCompletionFunc completes multiple local group names, excluding already-selected ones.
func multiLocalGroupCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := shared.ResolveClientForCompletion()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	groups, err := client.ListGroups(context.Background())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	selected := make(map[string]bool)
	for _, a := range args {
		selected[a] = true
	}
	var comps []string
	for _, g := range shared.FilterLocalGroups(groups) {
		if selected[g.Name] {
			continue
		}
		if strings.HasPrefix(g.Name, toComplete) {
			comps = append(comps, g.Name)
		}
	}
	return comps, cobra.ShellCompDirectiveNoFileComp
}

// allGroupCompletionFunc completes all group names (both local and oauth).
func allGroupCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client := shared.ResolveClientForCompletion()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	groups, err := client.ListGroups(context.Background())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var comps []string
	for _, g := range groups {
		if strings.HasPrefix(g.Name, toComplete) {
			comps = append(comps, g.Name)
		}
	}
	return comps, cobra.ShellCompDirectiveNoFileComp
}

// multiUserCompletionFunc returns a ValidArgsFunction that completes multiple user names filtered by role.
func multiUserCompletionFunc(roleFilter string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client := shared.ResolveClientForCompletion()
		if client == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		users, err := client.ListUsers(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		selected := make(map[string]bool)
		for _, a := range args {
			selected[a] = true
		}
		var comps []string
		for _, u := range users {
			if selected[u.Name] {
				continue
			}
			if roleFilter != "" && u.Role != roleFilter {
				continue
			}
			if strings.HasPrefix(u.Name, toComplete) {
				comps = append(comps, u.Name)
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	}
}
