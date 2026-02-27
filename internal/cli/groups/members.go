package groups

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var membersCmd = &cobra.Command{
	Use:               "members [group]",
	Short:             "Show group details and members",
	Aliases:           []string{"show"},
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: allGroupCompletionFunc,
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

		if len(groups) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No groups found.")
			return nil
		}

		var groupName string
		if len(args) > 0 {
			groupName = args[0]
		} else {
			options := make([]huh.Option[string], 0, len(groups))
			for _, g := range groups {
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", g.Name, groupType(g)), g.Name))
			}
			err := runSearchableSelect("Select group", options, &groupName)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
		}

		group, err := findGroupByName(groups, groupName)
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
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		// Pretty output
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Group: %s\n", fullGroup.Name)
		fmt.Fprintf(out, "Description: %s\n", fullGroup.Description)
		fmt.Fprintf(out, "Type: %s\n", groupType(*fullGroup))

		memberCount := len(members)
		if fullGroup.MemberCount != nil {
			memberCount = *fullGroup.MemberCount
		}
		fmt.Fprintf(out, "Members: %d\n", memberCount)

		if len(fullGroup.Permissions) > 0 && string(fullGroup.Permissions) != "null" {
			fmt.Fprintf(out, "Permissions: %s\n", string(fullGroup.Permissions))
		}

		if fullGroup.CreatedAt > 0 {
			fmt.Fprintf(out, "Created: %s\n", time.Unix(fullGroup.CreatedAt, 0).Format("2006-01-02 15:04:05"))
		}
		if fullGroup.UpdatedAt > 0 {
			fmt.Fprintf(out, "Updated: %s\n", time.Unix(fullGroup.UpdatedAt, 0).Format("2006-01-02 15:04:05"))
		}

		if len(members) > 0 {
			fmt.Fprintln(out)
			w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tEMAIL\tROLE")
			for _, m := range members {
				fmt.Fprintf(w, "%s\t%s\t%s\n", m.Name, m.Email, m.Role)
			}
			w.Flush()
		} else {
			fmt.Fprintln(out, "\nNo members in this group.")
		}

		return nil
	},
}
