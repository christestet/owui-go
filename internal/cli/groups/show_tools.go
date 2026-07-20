package groups

import (
	"fmt"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var showToolsCmd = &cobra.Command{
	Use:               "show-tools [group]",
	Short:             "List tools a group can read or write",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: allGroupCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		permission, _ := cmd.Flags().GetString("permission")
		switch permission {
		case "", "all", "read", "write":
		default:
			return fmt.Errorf("invalid permission %q: must be 'read', 'write', or 'all'", permission)
		}
		if permission == "" {
			permission = "all"
		}

		includePublic, _ := cmd.Flags().GetBool("include-public")

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
			if err := prompts.RunSearchableSelect("Select group", shared.GroupOptions(groups), &groupIdentifier); err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		group, err := shared.ResolveGroup(groups, groupIdentifier)
		if err != nil {
			return err
		}

		tools, err := client.ListTools(ctx)
		if err != nil {
			return err
		}

		var readTools, writeTools, publicTools []api.Tool
		for _, t := range tools {
			if len(t.AccessGrants) == 0 {
				if includePublic {
					publicTools = append(publicTools, t)
				}
				continue
			}
			for _, gr := range t.AccessGrants {
				if gr.PrincipalType != "group" || gr.PrincipalID != group.ID {
					continue
				}
				switch gr.Permission {
				case "read":
					readTools = append(readTools, t)
				case "write":
					writeTools = append(writeTools, t)
				}
			}
		}

		sortByName := func(s []api.Tool) {
			sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
		}
		sortByName(readTools)
		sortByName(writeTools)
		sortByName(publicTools)

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return renderShowToolsJSON(cmd, group, permission, includePublic, readTools, writeTools, publicTools)
		}
		return renderShowToolsPretty(cmd, group, permission, includePublic, readTools, writeTools, publicTools)
	},
}

func renderShowToolsJSON(cmd *cobra.Command, group *api.Group, permission string, includePublic bool, read, write, public []api.Tool) error {
	return renderGroupPermissionJSON(cmd, group, permission, includePublic, read, write, public)
}

func renderShowToolsPretty(cmd *cobra.Command, group *api.Group, permission string, includePublic bool, read, write, public []api.Tool) error {
	out := cmd.OutOrStdout()

	if _, err := fmt.Fprintf(out, "Group: %s (%s)\n", group.Name, group.ID); err != nil {
		return err
	}
	totalGrants := len(read) + len(write)
	if _, err := fmt.Fprintf(out, "Grants: %d tool(s)\n", totalGrants); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	showSection := func(label string, list []api.Tool) error {
		if _, err := fmt.Fprintf(out, "── %s ──\n", label); err != nil {
			return err
		}
		if len(list) == 0 {
			if _, err := fmt.Fprintln(out, "  (none)"); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			return nil
		}
		w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
		if _, err := fmt.Fprintln(w, "  NAME\tID\tUPDATED"); err != nil {
			return err
		}
		for _, t := range list {
			updated := "-"
			if t.UpdatedAt > 0 {
				updated = time.Unix(t.UpdatedAt, 0).Format("2006-01-02")
			}
			if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\n", t.Name, t.ID, updated); err != nil {
				return err
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}
		_, err := fmt.Fprintln(out)
		return err
	}

	if permission == "all" || permission == "read" {
		if err := showSection("Read", read); err != nil {
			return err
		}
	}
	if permission == "all" || permission == "write" {
		if err := showSection("Write", write); err != nil {
			return err
		}
	}
	if includePublic {
		if err := showSection("Public (no grants — visible to all)", public); err != nil {
			return err
		}
	}

	if totalGrants == 0 && !includePublic {
		_, err := fmt.Fprintf(out, "No tools grant access to group %q. Use --include-public to also list tools visible to everyone.\n", group.Name)
		return err
	}

	return nil
}

func init() {
	showToolsCmd.Flags().String("permission", "all", "filter by permission: read, write, or all")
	showToolsCmd.Flags().Bool("include-public", false, "also list public tools (no access grants)")
}
