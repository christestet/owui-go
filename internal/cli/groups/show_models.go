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

var showModelsCmd = &cobra.Command{
	Use:               "show-models [group]",
	Short:             "List models a group can read or write",
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

		models, err := client.ListModelsWithOptions(ctx, nil)
		if err != nil {
			return err
		}

		var readModels, writeModels, publicModels []api.ModelAccessResponse
		for _, m := range models {
			if len(m.AccessGrants) == 0 {
				if includePublic {
					publicModels = append(publicModels, m)
				}
				continue
			}
			for _, gr := range m.AccessGrants {
				if gr.PrincipalType != "group" || gr.PrincipalID != group.ID {
					continue
				}
				switch gr.Permission {
				case "read":
					readModels = append(readModels, m)
				case "write":
					writeModels = append(writeModels, m)
				}
			}
		}

		sortByName := func(s []api.ModelAccessResponse) {
			sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
		}
		sortByName(readModels)
		sortByName(writeModels)
		sortByName(publicModels)

		outputFormat, _ := cmd.Flags().GetString("output")

		if outputFormat == "json" {
			return renderShowModelsJSON(cmd, group, permission, includePublic, readModels, writeModels, publicModels)
		}

		return renderShowModelsPretty(cmd, group, permission, includePublic, readModels, writeModels, publicModels)
	},
}

func renderShowModelsJSON(cmd *cobra.Command, group *api.Group, permission string, includePublic bool, read, write, public []api.ModelAccessResponse) error {
	return renderGroupPermissionJSON(cmd, group, permission, includePublic, read, write, public)
}

func renderShowModelsPretty(cmd *cobra.Command, group *api.Group, permission string, includePublic bool, read, write, public []api.ModelAccessResponse) error {
	out := cmd.OutOrStdout()

	if _, err := fmt.Fprintf(out, "Group: %s (%s)\n", group.Name, group.ID); err != nil {
		return err
	}
	totalGrants := len(read) + len(write)
	if _, err := fmt.Fprintf(out, "Grants: %d model(s)\n", totalGrants); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	showSection := func(label string, list []api.ModelAccessResponse) error {
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
		if _, err := fmt.Fprintln(w, "  NAME\tID\tSTATUS\tUPDATED"); err != nil {
			return err
		}
		for _, m := range list {
			updated := "-"
			if m.UpdatedAt > 0 {
				updated = time.Unix(m.UpdatedAt, 0).Format("2006-01-02")
			}
			status := "disabled"
			if m.IsActive {
				status = "enabled"
			}
			if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", m.Name, m.ID, status, updated); err != nil {
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
		_, err := fmt.Fprintf(out, "No models grant access to group %q. Use --include-public to also list models visible to everyone.\n", group.Name)
		return err
	}

	return nil
}

func init() {
	showModelsCmd.Flags().String("permission", "all", "filter by permission: read, write, or all")
	showModelsCmd.Flags().Bool("include-public", false, "also list public models (no access grants)")
}
