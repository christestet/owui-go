package groups

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
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
			fmt.Fprintln(cmd.OutOrStdout(), "No local groups found.")
			return nil
		}

		// Resolve group
		groupName, _ := cmd.Flags().GetString("group")
		if groupName == "" {
			groupOptions := make([]huh.Option[string], 0, len(localGroups))
			for _, g := range localGroups {
				groupOptions = append(groupOptions, huh.NewOption(g.Name, g.Name))
			}
			err := prompts.RunSearchableSelect("Select group", groupOptions, &groupName)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		group, err := shared.FindGroupByName(localGroups, groupName)
		if err != nil {
			return err
		}

		// Fetch group members
		groupExport, err := client.ExportGroup(ctx, group.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch users of group %s: %w", group.Name, err)
		}
		if len(groupExport.UserIDs) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No users in group %s.\n", group.Name)
			return nil
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

		var selectedNames []string
		if len(args) > 0 {
			// Validate that provided names are members of the group
			allowed := make(map[string]struct{})
			for _, u := range groupUsers {
				allowed[u.Name] = struct{}{}
			}
			for _, name := range args {
				if _, ok := allowed[name]; ok {
					selectedNames = append(selectedNames, name)
				}
			}
			if len(selectedNames) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No valid users specified (must be members of the group).")
				return nil
			}
		} else {
			if len(groupUsers) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No users found in group %s.\n", group.Name)
				return nil
			}
			options := make([]huh.Option[string], 0, len(groupUsers))
			for _, u := range groupUsers {
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", u.Name, u.Email), u.Name))
			}
			err := prompts.RunSearchableMultiSelect("Select users to remove from group", options, &selectedNames)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
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
			u, err := shared.FindUserByName(groupUsers, name)
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

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully removed %s from group '%s'\n", strings.Join(resolvedNames, ", "), group.Name)
		return nil
	},
}

func init() {
	removeUsersCmd.Flags().String("group", "", "group name to remove users from")
	_ = removeUsersCmd.RegisterFlagCompletionFunc("group", localGroupCompletionFunc)
}
