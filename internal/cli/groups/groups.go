package groups

import (
	"context"
	"fmt"
	"strings"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/config"
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
	groupsCmd.AddCommand(addUsersCmd)
	groupsCmd.AddCommand(removeUsersCmd)

	rootCmd.AddCommand(groupsCmd)
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

// isOAuthGroup returns true if the group was auto-created via OAuth.
func isOAuthGroup(g api.Group) bool {
	return strings.HasPrefix(g.Description, "Group ") &&
		strings.HasSuffix(g.Description, " created automatically via OAuth.")
}

// groupType returns "oauth" or "local" for a group.
func groupType(g api.Group) string {
	if isOAuthGroup(g) {
		return "oauth"
	}
	return "local"
}

// filterLocalGroups returns only non-OAuth groups.
func filterLocalGroups(groups []api.Group) []api.Group {
	var local []api.Group
	for _, g := range groups {
		if !isOAuthGroup(g) {
			local = append(local, g)
		}
	}
	return local
}

// filterOAuthGroups returns only OAuth groups.
func filterOAuthGroups(groups []api.Group) []api.Group {
	var oauth []api.Group
	for _, g := range groups {
		if isOAuthGroup(g) {
			oauth = append(oauth, g)
		}
	}
	return oauth
}

// findGroupByName looks up a group by name from a group list.
func findGroupByName(groups []api.Group, name string) (*api.Group, error) {
	for i := range groups {
		if groups[i].Name == name {
			return &groups[i], nil
		}
	}
	return nil, fmt.Errorf("group %q not found", name)
}

// findUserByName looks up a user by name from a user list.
func findUserByName(users []api.User, name string) (*api.User, error) {
	for i := range users {
		if users[i].Name == name {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("user %q not found", name)
}

// filterUsersByRole returns only users with the given role.
func filterUsersByRole(users []api.User, role string) []api.User {
	var filtered []api.User
	for _, u := range users {
		if u.Role == role {
			filtered = append(filtered, u)
		}
	}
	return filtered
}

// localGroupCompletionFunc completes only local group names.
func localGroupCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := resolveClientForCompletion()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	groups, err := client.ListGroups(context.Background())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var comps []string
	for _, g := range filterLocalGroups(groups) {
		if strings.HasPrefix(g.Name, toComplete) {
			comps = append(comps, g.Name)
		}
	}
	return comps, cobra.ShellCompDirectiveNoFileComp
}

// multiLocalGroupCompletionFunc completes multiple local group names, excluding already-selected ones.
func multiLocalGroupCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := resolveClientForCompletion()
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
	for _, g := range filterLocalGroups(groups) {
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
	client := resolveClientForCompletion()
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
		client := resolveClientForCompletion()
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
