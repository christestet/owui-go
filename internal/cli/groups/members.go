package groups

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var membersCmd = &cobra.Command{
	Use:               "members [group]",
	Short:             "Show group details and members",
	Aliases:           []string{"show"},
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: allGroupCompletionFunc,
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

		if len(groups) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No groups found.")
			return err
		}

		var groupIdentifier string
		if len(args) > 0 {
			groupIdentifier = args[0]
		} else {
			err := prompts.RunSearchableSelect("Select group", shared.GroupOptions(groups), &groupIdentifier)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		group, err := shared.ResolveGroup(groups, groupIdentifier)
		if err != nil {
			return err
		}

		// Fetch full group details
		fullGroup, err := client.GetGroup(ctx, group.ID)
		if err != nil {
			return err
		}

		// Fetch members
		members, err := client.GetGroupMembers(ctx, group.ID)
		if err != nil {
			return err
		}

		outputFormat, _ := cmd.Flags().GetString("output")

		if outputFormat == "json" {
			result := struct {
				Group   interface{} `json:"group"`
				Members interface{} `json:"members"`
			}{
				Group:   fullGroup,
				Members: members,
			}
			b, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return err
		}

		// Pretty output
		out := cmd.OutOrStdout()
		if _, err := fmt.Fprintf(out, "Group: %s\n", fullGroup.Name); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "Description: %s\n", fullGroup.Description); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "Type: %s\n", groupType(*fullGroup)); err != nil {
			return err
		}

		memberCount := len(members)
		if fullGroup.MemberCount != nil {
			memberCount = *fullGroup.MemberCount
		}
		if _, err := fmt.Fprintf(out, "Members: %d\n", memberCount); err != nil {
			return err
		}

		if len(fullGroup.Permissions) > 0 && string(fullGroup.Permissions) != "null" {
			if _, err := fmt.Fprintf(out, "Permissions: %s\n", string(fullGroup.Permissions)); err != nil {
				return err
			}
		}

		if fullGroup.CreatedAt > 0 {
			if _, err := fmt.Fprintf(out, "Created: %s\n", time.Unix(fullGroup.CreatedAt, 0).Format("2006-01-02 15:04:05")); err != nil {
				return err
			}
		}
		if fullGroup.UpdatedAt > 0 {
			if _, err := fmt.Fprintf(out, "Updated: %s\n", time.Unix(fullGroup.UpdatedAt, 0).Format("2006-01-02 15:04:05")); err != nil {
				return err
			}
		}

		if len(members) > 0 {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			if _, err := fmt.Fprintln(w, "NAME\tEMAIL\tROLE"); err != nil {
				return err
			}
			for _, m := range members {
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", m.Name, m.Email, m.Role); err != nil {
					return err
				}
			}
			if err := w.Flush(); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(out, "\nNo members in this group."); err != nil {
				return err
			}
		}

		return nil
	},
}
