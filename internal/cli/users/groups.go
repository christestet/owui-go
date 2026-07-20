package users

import (
	"context"
	"fmt"
	"strings"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

// localGroupCompletionFunc returns a flag completion function for local group identifiers.
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

// --- add-to-group command ---

var addToGroupCmd = &cobra.Command{
	Use:               "add-to-group [user...]",
	Short:             "Add user(s) to a group",
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: multiUserCompletionFunc("user"),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		allUsers, err := client.ListUsers(ctx)
		if err != nil {
			return err
		}

		groups, err := client.ListGroups(ctx)
		if err != nil {
			return err
		}
		localGroups := shared.FilterLocalGroups(groups)

		if len(localGroups) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No local groups found.")
			return err
		}

		// Resolve group
		groupIdentifier, _ := cmd.Flags().GetString("group")
		if groupIdentifier == "" {
			err := prompts.RunSearchableSelect("Select group", shared.GroupOptions(localGroups), &groupIdentifier)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		group, err := shared.ResolveGroup(localGroups, groupIdentifier)
		if err != nil {
			return err
		}
		var resolvedUsers []api.User
		if len(args) > 0 {
			resolvedUsers, err = shared.ResolveUsers(allUsers, args)
			if err != nil {
				return err
			}
			for _, user := range resolvedUsers {
				if user.Role != "user" {
					return fmt.Errorf("user %q has role %q; only users with role 'user' can be added to groups", user.Name, user.Role)
				}
			}
		}

		groupExport, err := client.ExportGroup(ctx, group.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch users of group %s: %w", group.Name, err)
		}
		existingIDs := make(map[string]struct{}, len(groupExport.UserIDs))
		for _, id := range groupExport.UserIDs {
			existingIDs[id] = struct{}{}
		}
		eligibleUsers := make([]api.User, 0, len(allUsers))
		for _, user := range allUsers {
			if user.Role != "user" {
				continue
			}
			if _, exists := existingIDs[user.ID]; exists {
				continue
			}
			eligibleUsers = append(eligibleUsers, user)
		}

		if len(args) == 0 {
			if len(eligibleUsers) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No users available to add (only users with role 'user' who are not already members can be added).")
				return err
			}
			var selectedIdentifiers []string
			if err := prompts.RunSearchableMultiSelect("Select users to add to group", shared.UserOptions(eligibleUsers), &selectedIdentifiers); err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
			resolvedUsers, err = shared.ResolveUsers(eligibleUsers, selectedIdentifiers)
			if err != nil {
				return err
			}
		}
		userIDs := make([]string, 0, len(resolvedUsers))
		resolvedNames := make([]string, 0, len(resolvedUsers))
		for _, u := range resolvedUsers {
			if _, exists := existingIDs[u.ID]; exists {
				continue
			}
			userIDs = append(userIDs, u.ID)
			resolvedNames = append(resolvedNames, shared.UserLabel(u))
			existingIDs[u.ID] = struct{}{}
		}
		if len(userIDs) == 0 {
			if len(args) > 0 {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "All selected users are already members of group '%s'.\n", group.Name)
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No users selected.")
			return err
		}

		confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Confirm adding %d user(s) to group '%s'?", len(resolvedNames), group.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return err
		}

		if err := client.AddUsersToGroup(ctx, group.ID, userIDs); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Successfully added %s to group %s\n", strings.Join(resolvedNames, ", "), group.Name)
		return err
	},
}

// --- remove-from-group command ---

var removeFromGroupCmd = &cobra.Command{
	Use:   "remove-from-group [user...]",
	Short: "Remove user(s) from a group",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		allUsers, err := client.ListUsers(ctx)
		if err != nil {
			return err
		}

		groups, err := client.ListGroups(ctx)
		if err != nil {
			return err
		}
		// Nur lokale Gruppen zulassen (keine OAuth-Gruppen)
		localGroups := shared.FilterLocalGroups(groups)
		if len(localGroups) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No local groups found.")
			return err
		}

		// --- Gruppe auswählen ---
		groupIdentifier, _ := cmd.Flags().GetString("group")
		if groupIdentifier == "" {
			err := prompts.RunSearchableSelect("Select group", shared.GroupOptions(localGroups), &groupIdentifier)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		group, err := shared.ResolveGroup(localGroups, groupIdentifier)
		if err != nil {
			return err
		}

		// --- Nutzer der Gruppe ermitteln ---
		groupExport, err := client.ExportGroup(ctx, group.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch users of group %s: %w", group.Name, err)
		}
		if len(groupExport.UserIDs) == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "No users in group %s.\n", group.Name)
			return err
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
		eligibleUsers := shared.FilterUsersByRole(groupUsers, "user")

		var selectedIdentifiers []string
		if len(args) > 0 {
			selectedIdentifiers = args
		} else {
			if len(eligibleUsers) == 0 {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "No eligible users found in group %s.\n", group.Name)
				return err
			}
			err := prompts.RunSearchableMultiSelect("Select users to remove from group", shared.UserOptions(eligibleUsers), &selectedIdentifiers)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		if len(selectedIdentifiers) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No users selected.")
			return err
		}

		// Resolve user identifiers to IDs.
		resolvedUsers, err := shared.ResolveUsers(eligibleUsers, selectedIdentifiers)
		if err != nil {
			return fmt.Errorf("user must be a member of group %q with role 'user': %w", group.Name, err)
		}
		userIDs := make([]string, 0, len(resolvedUsers))
		resolvedNames := make([]string, 0, len(resolvedUsers))
		for _, u := range resolvedUsers {
			userIDs = append(userIDs, u.ID)
			resolvedNames = append(resolvedNames, shared.UserLabel(u))
		}

		confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Confirm removing %d user(s) from group '%s'?", len(resolvedNames), group.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return err
		}

		if err := client.RemoveUsersFromGroup(ctx, group.ID, userIDs); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Successfully removed %s from group %s\n", strings.Join(resolvedNames, ", "), group.Name)
		return err
	},
}

func init() {
	addToGroupCmd.Flags().String("group", "", "group name or ID to add users to")
	_ = addToGroupCmd.RegisterFlagCompletionFunc("group", localGroupCompletionFunc)

	removeFromGroupCmd.Flags().String("group", "", "group name or ID to remove users from")
	_ = removeFromGroupCmd.RegisterFlagCompletionFunc("group", localGroupCompletionFunc)
}
