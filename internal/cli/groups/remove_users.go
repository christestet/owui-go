package groups

import (
	"fmt"
	"strings"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var removeUsersCmd = &cobra.Command{
	Use:   "remove-users [user...]",
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
		localGroups := shared.FilterLocalGroups(groups)

		if len(localGroups) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No local groups found.")
			return err
		}

		// Resolve group
		groupName, _ := cmd.Flags().GetString("group")
		if groupName == "" {
			groupOptions := shared.GroupOptions(localGroups)
			err := prompts.RunSearchableSelect("Select group", groupOptions, &groupName)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		group, err := shared.ResolveGroup(localGroups, groupName)
		if err != nil {
			return err
		}

		// Fetch group members
		groupExport, err := client.ExportGroup(ctx, group.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch users of group %s: %w", group.Name, err)
		}
		if len(groupExport.UserIDs) == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "No users in group %s.\n", group.Name)
			return err
		}

		// Build user lookup
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

		var selectedIdentifiers []string
		if len(args) > 0 {
			selectedIdentifiers = args
		} else {
			if len(groupUsers) == 0 {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "No users found in group %s.\n", group.Name)
				return err
			}
			err := prompts.RunSearchableMultiSelect("Select users to remove from group", shared.UserOptions(groupUsers), &selectedIdentifiers)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		if len(selectedIdentifiers) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No users selected.")
			return err
		}

		// Resolve user identifiers to IDs.
		resolvedUsers, err := shared.ResolveUsers(groupUsers, selectedIdentifiers)
		if err != nil {
			return fmt.Errorf("user must be a member of group %q: %w", group.Name, err)
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

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Successfully removed %s from group '%s'\n", strings.Join(resolvedNames, ", "), group.Name)
		return err
	},
}

func init() {
	removeUsersCmd.Flags().String("group", "", "group name or ID to remove users from")
	_ = removeUsersCmd.RegisterFlagCompletionFunc("group", localGroupCompletionFunc)
}
