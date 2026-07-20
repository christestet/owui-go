package groups

import (
	"context"

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
	groupsCmd.AddCommand(showPermissionsCmd)
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

// localGroupCompletionFunc completes only local group identifiers.
func localGroupCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := shared.ResolveClientForCompletion(cmd)
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	groups, err := client.ListGroups(context.Background())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return shared.GroupCompletions(shared.FilterLocalGroups(groups), nil, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// multiLocalGroupCompletionFunc completes multiple local group identifiers, excluding selected ones.
func multiLocalGroupCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := shared.ResolveClientForCompletion(cmd)
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	groups, err := client.ListGroups(context.Background())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return shared.GroupCompletions(shared.FilterLocalGroups(groups), args, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// allGroupCompletionFunc completes all group identifiers (both local and oauth).
func allGroupCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client := shared.ResolveClientForCompletion(cmd)
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	groups, err := client.ListGroups(context.Background())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return shared.GroupCompletions(groups, nil, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// multiUserCompletionFunc completes multiple user identifiers filtered by role.
func multiUserCompletionFunc(roleFilter string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client := shared.ResolveClientForCompletion(cmd)
		if client == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		users, err := client.ListUsers(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var eligible []api.User
		for _, u := range users {
			if roleFilter != "" && u.Role != roleFilter {
				continue
			}
			eligible = append(eligible, u)
		}
		return shared.UserCompletions(eligible, args, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}
