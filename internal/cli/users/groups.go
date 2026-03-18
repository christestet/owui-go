package users

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/spf13/cobra"
)

// isOAuthGroup returns true if the group was auto-created via OAuth.
func isOAuthGroup(g api.Group) bool {
	return strings.HasPrefix(g.Description, "Group ") &&
		strings.HasSuffix(g.Description, " created automatically via OAuth.")
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

// findGroupByName looks up a group by name from a group list.
func findGroupByName(groups []api.Group, name string) (*api.Group, error) {
	for i := range groups {
		if groups[i].Name == name {
			return &groups[i], nil
		}
	}
	return nil, fmt.Errorf("group %q not found", name)
}

// localGroupCompletionFunc returns a flag completion function for local group names.
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

// --- add-to-group command ---

var addToGroupCmd = &cobra.Command{
	Use:               "add-to-group [user...]",
	Short:             "Add user(s) to a group",
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: multiUserCompletionFunc("user"),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := resolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		allUsers, err := client.ListUsers(ctx)
		if err != nil {
			return err
		}

		groups, err := client.ListGroups(ctx)
		if err != nil {
			return err
		}
		localGroups := filterLocalGroups(groups)

		if len(localGroups) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No local groups found.")
			return nil
		}

		// Filter to only role=user
		eligibleUsers := filterUsersByRole(allUsers, "user")

		var selectedNames []string
		if len(args) > 0 {
			selectedNames = args
		} else {
			// Interactive multi-select
			if len(eligibleUsers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No eligible users found (only users with role 'user' can be added to groups).")
				return nil
			}
			options := make([]huh.Option[string], 0, len(eligibleUsers))
			for _, u := range eligibleUsers {
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", u.Name, u.Email), u.Name))
			}
			err := runSearchableMultiSelect("Select users to add to group", options, &selectedNames)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
		}

		if len(selectedNames) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No users selected.")
			return nil
		}

		// Resolve group
		groupName, _ := cmd.Flags().GetString("group")
		if groupName == "" {
			groupOptions := make([]huh.Option[string], 0, len(localGroups))
			for _, g := range localGroups {
				groupOptions = append(groupOptions, huh.NewOption(g.Name, g.Name))
			}
			err := runSearchableSelect("Select group", groupOptions, &groupName)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
		}

		group, err := findGroupByName(localGroups, groupName)
		if err != nil {
			return err
		}

		// Resolve user names to IDs
		var userIDs []string
		var resolvedNames []string
		for _, name := range selectedNames {
			u, err := findUserByName(allUsers, name)
			if err != nil {
				return err
			}
			if u.Role != "user" {
				return fmt.Errorf("user %q has role %q; only users with role 'user' can be added to groups", u.Name, u.Role)
			}
			userIDs = append(userIDs, u.ID)
			resolvedNames = append(resolvedNames, u.Name)
		}

		confirmed, err := prompts.ConfirmYN(fmt.Sprintf("Confirm adding %d user(s) to group '%s'?", len(resolvedNames), group.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		if err := client.AddUsersToGroup(ctx, group.ID, userIDs); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully added %s to group %s\n", strings.Join(resolvedNames, ", "), group.Name)
		return nil
	},
}

// --- remove-from-group command ---

var removeFromGroupCmd = &cobra.Command{
	Use:   "remove-from-group [user...]",
	Short: "Remove user(s) from a group",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := resolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		allUsers, err := client.ListUsers(ctx)
		if err != nil {
			return err
		}

		groups, err := client.ListGroups(ctx)
		if err != nil {
			return err
		}
		// Nur lokale Gruppen zulassen (keine OAuth-Gruppen)
		localGroups := filterLocalGroups(groups)
		if len(localGroups) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No local groups found.")
			return nil
		}

		// --- Gruppe auswählen ---
		groupName, _ := cmd.Flags().GetString("group")
		if groupName == "" {
			groupOptions := make([]huh.Option[string], 0, len(localGroups))
			for _, g := range localGroups {
				groupOptions = append(groupOptions, huh.NewOption(g.Name, g.Name))
			}
			err := runSearchableSelect("Select group", groupOptions, &groupName)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
		}

		group, err := findGroupByName(localGroups, groupName)
		if err != nil {
			return err
		}

		// --- Nutzer der Gruppe ermitteln ---
		groupExport, err := client.ExportGroup(ctx, group.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch users of group %s: %w", group.Name, err)
		}
		if len(groupExport.UserIDs) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No users in group %s.\n", group.Name)
			return nil
		}
		// Nutzerobjekte zu den IDs suchen
		userByID := make(map[string]api.User)
		for _, u := range allUsers {
			userByID[u.ID] = u
		}
		var groupUsers []api.User
		for _, uid := range groupExport.UserIDs {
			if u, ok := userByID[uid]; ok {
				groupUsers = append(groupUsers, u)
			}
		}
		// Nur Nutzer mit Rolle "user" aus der Gruppe
		eligibleUsers := filterUsersByRole(groupUsers, "user")

		var selectedNames []string
		if len(args) > 0 {
			// Nur Namen zulassen, die in eligibleUsers sind
			allowed := make(map[string]struct{})
			for _, u := range eligibleUsers {
				allowed[u.Name] = struct{}{}
			}
			for _, name := range args {
				if _, ok := allowed[name]; ok {
					selectedNames = append(selectedNames, name)
				}
			}
			if len(selectedNames) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No valid users specified (must be members of the group with role 'user').")
				return nil
			}
		} else {
			if len(eligibleUsers) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No eligible users found in group %s.\n", group.Name)
				return nil
			}
			options := make([]huh.Option[string], 0, len(eligibleUsers))
			for _, u := range eligibleUsers {
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", u.Name, u.Email), u.Name))
			}
			err := runSearchableMultiSelect("Select users to remove from group", options, &selectedNames)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
		}

		if len(selectedNames) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No users selected.")
			return nil
		}

		// Resolve user names to IDs
		var userIDs []string
		var resolvedNames []string
		for _, name := range selectedNames {
			u, err := findUserByName(groupUsers, name)
			if err != nil {
				return err
			}
			userIDs = append(userIDs, u.ID)
			resolvedNames = append(resolvedNames, u.Name)
		}

		confirmed, err := prompts.ConfirmYN(fmt.Sprintf("Confirm removing %d user(s) from group '%s'?", len(resolvedNames), group.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		if err := client.RemoveUsersFromGroup(ctx, group.ID, userIDs); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully removed %s from group %s\n", strings.Join(resolvedNames, ", "), group.Name)
		return nil
	},
}

func init() {
	addToGroupCmd.Flags().String("group", "", "group name to add users to")
	_ = addToGroupCmd.RegisterFlagCompletionFunc("group", localGroupCompletionFunc)

	removeFromGroupCmd.Flags().String("group", "", "group name to remove users from")
	_ = removeFromGroupCmd.RegisterFlagCompletionFunc("group", localGroupCompletionFunc)
}
