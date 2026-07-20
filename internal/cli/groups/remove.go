package groups

import (
	"fmt"
	"strings"

	"github.com/christestet/owui-go/internal/api"
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
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No local groups found.")
			return err
		}

		var selectedIdentifiers []string
		if len(args) > 0 {
			selectedIdentifiers = args
		} else {
			err := prompts.RunSearchableMultiSelect("Select groups to delete", shared.GroupOptions(localGroups), &selectedIdentifiers)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		if len(selectedIdentifiers) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No groups selected.")
			return err
		}

		resolvedGroups := make([]api.Group, 0, len(selectedIdentifiers))
		seen := make(map[string]struct{}, len(selectedIdentifiers))
		for _, identifier := range selectedIdentifiers {
			group, err := shared.ResolveGroup(localGroups, identifier)
			if err != nil {
				return err
			}
			if _, ok := seen[group.ID]; ok {
				continue
			}
			seen[group.ID] = struct{}{}
			resolvedGroups = append(resolvedGroups, *group)
		}
		selectedNames := make([]string, 0, len(resolvedGroups))
		for _, group := range resolvedGroups {
			selectedNames = append(selectedNames, group.Name)
		}

		confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Confirm deleting %d group(s): %s?", len(selectedNames), strings.Join(selectedNames, ", ")))
		if err != nil {
			return err
		}
		if !confirmed {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return err
		}

		for _, group := range resolvedGroups {
			if err := client.DeleteGroup(ctx, group.ID); err != nil {
				return fmt.Errorf("failed to delete group %q: %w", group.Name, err)
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted group '%s'\n", group.Name); err != nil {
				return err
			}
		}

		return nil
	},
}
