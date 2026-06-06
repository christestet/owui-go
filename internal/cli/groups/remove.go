package groups

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:               "remove [group...]",
	Short:             "Remove one or more groups",
	Aliases:           []string{"rm"},
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: multiLocalGroupCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		groups, err := client.ListGroups(ctx)
		if err != nil {
			return err
		}
		localGroups := shared.FilterLocalGroups(groups)

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
			err := prompts.RunSearchableMultiSelect("Select groups to delete", options, &selectedNames)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		if len(selectedNames) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No groups selected.")
			return nil
		}

		// Validate all group names exist
		for _, name := range selectedNames {
			if _, err := shared.FindGroupByName(localGroups, name); err != nil {
				return err
			}
		}

		confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Confirm deleting %d group(s): %s?", len(selectedNames), strings.Join(selectedNames, ", ")))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		for _, name := range selectedNames {
			group, _ := shared.FindGroupByName(localGroups, name)
			if err := client.DeleteGroup(ctx, group.ID); err != nil {
				return fmt.Errorf("failed to delete group %q: %w", name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted group '%s'\n", name)
		}

		return nil
	},
}
