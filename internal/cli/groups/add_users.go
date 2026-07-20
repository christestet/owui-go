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

var addUsersCmd = &cobra.Command{
	Use:               "add-users [user...]",
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

		groupExport, err := client.ExportGroup(ctx, group.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch users of group %s: %w", group.Name, err)
		}

		eligibleUsers := filterAddableUsers(allUsers, groupExport.UserIDs)

		var userIDs []string
		var resolvedNames []string
		if len(args) > 0 {
			existingUserIDs := makeStringSet(groupExport.UserIDs)
			for _, identifier := range args {
				u, err := shared.ResolveUser(allUsers, identifier)
				if err != nil {
					return err
				}
				if u.Role != "user" {
					return fmt.Errorf("user %q has role %q; only users with role 'user' can be added to groups", u.Name, u.Role)
				}
				if _, exists := existingUserIDs[u.ID]; exists {
					continue
				}
				userIDs = append(userIDs, u.ID)
				resolvedNames = append(resolvedNames, u.Name)
				existingUserIDs[u.ID] = struct{}{}
			}
		} else {
			if len(eligibleUsers) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No users available to add (only users with role 'user' who are not already members can be added).")
				return err
			}

			var selectedUserIDs []string
			err := prompts.RunSearchableMultiSelect("Select users to add", userIDOptions(eligibleUsers), &selectedUserIDs)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}

			usersByID := make(map[string]api.User, len(eligibleUsers))
			for _, u := range eligibleUsers {
				usersByID[u.ID] = u
			}
			selected := makeStringSet(nil)
			for _, userID := range selectedUserIDs {
				if _, exists := selected[userID]; exists {
					continue
				}
				u, exists := usersByID[userID]
				if !exists {
					return fmt.Errorf("selected user ID %q not found", userID)
				}
				userIDs = append(userIDs, u.ID)
				resolvedNames = append(resolvedNames, u.Name)
				selected[userID] = struct{}{}
			}
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

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Successfully added %s to group '%s'\n", strings.Join(resolvedNames, ", "), group.Name)
		return err
	},
}

func init() {
	addUsersCmd.Flags().String("group", "", "group name or ID to add users to")
	_ = addUsersCmd.RegisterFlagCompletionFunc("group", localGroupCompletionFunc)
}

func filterAddableUsers(users []api.User, existingUserIDs []string) []api.User {
	existing := makeStringSet(existingUserIDs)
	addable := make([]api.User, 0, len(users))
	for _, u := range users {
		if u.Role != "user" {
			continue
		}
		if _, exists := existing[u.ID]; exists {
			continue
		}
		addable = append(addable, u)
	}
	return addable
}

func userIDOptions(users []api.User) []huh.Option[string] {
	return shared.UserOptions(users)
}

func makeStringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}
