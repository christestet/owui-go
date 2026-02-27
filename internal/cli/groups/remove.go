package groups

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:               "remove [group...]",
	Short:             "Remove one or more groups",
	Aliases:           []string{"rm"},
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: multiLocalGroupCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := resolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
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

		var selectedNames []string
		if len(args) > 0 {
			selectedNames = args
		} else {
			// Interactive multi-select
			options := make([]huh.Option[string], 0, len(localGroups))
			for _, g := range localGroups {
				members := "-"
				if g.MemberCount != nil {
					members = fmt.Sprintf("%d", *g.MemberCount)
				}
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s members)", g.Name, members), g.Name))
			}
			err := runSearchableMultiSelect("Select groups to delete", options, &selectedNames)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
		}

		if len(selectedNames) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No groups selected.")
			return nil
		}

		// Validate all group names exist
		for _, name := range selectedNames {
			if _, err := findGroupByName(localGroups, name); err != nil {
				return err
			}
		}

		var confirmed bool
		err = huh.NewConfirm().
			Title(fmt.Sprintf("Confirm deleting %d group(s): %s?", len(selectedNames), strings.Join(selectedNames, ", "))).
			Value(&confirmed).
			Run()
		if err != nil {
			return wrapInteractiveCancelled(err)
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		for _, name := range selectedNames {
			group, _ := findGroupByName(localGroups, name)
			if err := client.DeleteGroup(ctx, group.ID); err != nil {
				return fmt.Errorf("failed to delete group %q: %w", name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted group '%s'\n", name)
		}

		return nil
	},
}
