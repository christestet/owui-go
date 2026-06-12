package groups

import (
	"encoding/json"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all groups",
	Aliases: []string{"ls"},
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

		filterType, _ := cmd.Flags().GetString("filter")
		switch filterType {
		case "local":
			groups = shared.FilterLocalGroups(groups)
		case "oauth":
			groups = filterOAuthGroups(groups)
		case "":
			// no filter
		default:
			return fmt.Errorf("invalid filter %q: must be 'local' or 'oauth'", filterType)
		}

		sort.Slice(groups, func(i, j int) bool {
			return groups[i].Name < groups[j].Name
		})

		outputFormat, _ := cmd.Flags().GetString("output")

		if outputFormat == "json" {
			b, err := json.MarshalIndent(groups, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return err
		}

		if len(groups) == 0 {
			if filterType != "" {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "No groups found matching filter %q.\n", filterType)
				return err
			} else {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No groups found.")
				return err
			}
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		if _, err := fmt.Fprintln(w, "NAME\tDESCRIPTION\tMEMBERS\tTYPE"); err != nil {
			return err
		}
		for _, g := range groups {
			members := "-"
			if g.MemberCount != nil {
				members = fmt.Sprintf("%d", *g.MemberCount)
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", g.Name, g.Description, members, groupType(g)); err != nil {
				return err
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}

		if filterType != "" {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d group(s) matching filter %q.\n", len(groups), filterType)
			return err
		} else {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d group(s).\n", len(groups))
			return err
		}
	},
}
