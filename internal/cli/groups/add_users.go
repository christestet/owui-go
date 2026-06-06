package groups

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
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

		// Filter to only role=user
		eligibleUsers := shared.FilterUsersByRole(allUsers, "user")

		var selectedNames []string
		if len(args) > 0 {
			selectedNames = args
		} else {
			if len(eligibleUsers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No eligible users found (only users with role 'user' can be added to groups).")
				return nil
			}
			options := make([]huh.Option[string], 0, len(eligibleUsers))
			for _, u := range eligibleUsers {
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", u.Name, u.Email), u.Name))
			}
			err := prompts.RunSearchableMultiSelect("Select users to add", options, &selectedNames)
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
			u, err := shared.FindUserByName(allUsers, name)
			if err != nil {
				return err
			}
			if u.Role != "user" {
				return fmt.Errorf("user %q has role %q; only users with role 'user' can be added to groups", u.Name, u.Role)
			}
			userIDs = append(userIDs, u.ID)
			resolvedNames = append(resolvedNames, u.Name)
		}

		confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Confirm adding %d user(s) to group '%s'?", len(resolvedNames), group.Name))
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

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully added %s to group '%s'\n", strings.Join(resolvedNames, ", "), group.Name)
		return nil
	},
}

func init() {
	addUsersCmd.Flags().String("group", "", "group name to add users to")
	_ = addUsersCmd.RegisterFlagCompletionFunc("group", localGroupCompletionFunc)
}
