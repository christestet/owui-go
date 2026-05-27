package groups

import (
	"encoding/json"
	"fmt"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/huh"
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
			if err := prompts.RunSearchableSelect("Select group", options, &groupName); err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		group, err := shared.FindGroupByName(groups, groupName)
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
	type groupRef struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	out := struct {
		Group  groupRef                     `json:"group"`
		Read   []api.ModelAccessResponse    `json:"read,omitempty"`
		Write  []api.ModelAccessResponse    `json:"write,omitempty"`
		Public []api.ModelAccessResponse    `json:"public,omitempty"`
	}{
		Group: groupRef{ID: group.ID, Name: group.Name},
	}
	if permission == "all" || permission == "read" {
		out.Read = read
	}
	if permission == "all" || permission == "write" {
		out.Write = write
	}
	if includePublic {
		out.Public = public
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}

func renderShowModelsPretty(cmd *cobra.Command, group *api.Group, permission string, includePublic bool, read, write, public []api.ModelAccessResponse) error {
	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "Group: %s (%s)\n", group.Name, group.ID)
	totalGrants := len(read) + len(write)
	fmt.Fprintf(out, "Grants: %d model(s)\n", totalGrants)
	fmt.Fprintln(out)

	showSection := func(label string, list []api.ModelAccessResponse) {
		fmt.Fprintf(out, "── %s ──\n", label)
		if len(list) == 0 {
			fmt.Fprintln(out, "  (none)")
			fmt.Fprintln(out)
			return
		}
		w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "  NAME\tID\tSTATUS\tUPDATED")
		for _, m := range list {
			updated := "-"
			if m.UpdatedAt > 0 {
				updated = time.Unix(m.UpdatedAt, 0).Format("2006-01-02")
			}
			status := "disabled"
			if m.IsActive {
				status = "enabled"
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", m.Name, m.ID, status, updated)
		}
		w.Flush()
		fmt.Fprintln(out)
	}

	if permission == "all" || permission == "read" {
		showSection("Read", read)
	}
	if permission == "all" || permission == "write" {
		showSection("Write", write)
	}
	if includePublic {
		showSection("Public (no grants — visible to all)", public)
	}

	if totalGrants == 0 && !includePublic {
		fmt.Fprintf(out, "No models grant access to group %q. Use --include-public to also list models visible to everyone.\n", group.Name)
	}

	return nil
}

func init() {
	showModelsCmd.Flags().String("permission", "all", "filter by permission: read, write, or all")
	showModelsCmd.Flags().Bool("include-public", false, "also list public models (no access grants)")
}
