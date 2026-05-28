package groups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var showPermissionsCmd = &cobra.Command{
	Use:               "show-permissions [group]",
	Short:             "Show group permissions",
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

		fullGroup, err := client.GetGroup(ctx, group.ID)
		if err != nil {
			return err
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return renderShowPermissionsJSON(cmd, fullGroup)
		}

		return renderShowPermissionsPretty(cmd, fullGroup)
	},
}

func renderShowPermissionsJSON(cmd *cobra.Command, group *api.Group) error {
	type groupRef struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	out := struct {
		Group       groupRef        `json:"group"`
		Permissions json.RawMessage `json:"permissions"`
	}{
		Group:       groupRef{ID: group.ID, Name: group.Name},
		Permissions: normalizedPermissions(group.Permissions),
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}

func renderShowPermissionsPretty(cmd *cobra.Command, group *api.Group) error {
	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "Group: %s (%s)\n", group.Name, group.ID)
	fmt.Fprintln(out, "Permissions:")

	permissions := normalizedPermissions(group.Permissions)
	if string(permissions) == "null" {
		fmt.Fprintln(out, "No permissions set.")
		return nil
	}

	rows, err := permissionRows(permissions)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Fprintln(out, "No permissions set.")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "PERMISSION\tVALUE")
	for _, row := range rows {
		fmt.Fprintf(w, "%s\t%s\n", row.Path, row.Value)
	}
	w.Flush()
	return nil
}

func normalizedPermissions(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("null")
	}
	return json.RawMessage(trimmed)
}

type permissionRow struct {
	Path  string
	Value string
}

func permissionRows(raw json.RawMessage) ([]permissionRow, error) {
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("decoding group permissions: %w", err)
	}

	var rows []permissionRow
	flattenPermissionValue(nil, value, &rows)
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Path < rows[j].Path
	})
	return rows, nil
}

func flattenPermissionValue(path []string, value interface{}, rows *[]permissionRow) {
	switch v := value.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			if len(path) == 0 {
				return
			}
			*rows = append(*rows, permissionRow{Path: strings.Join(path, "."), Value: "{}"})
			return
		}
		for key, child := range v {
			flattenPermissionValue(append(path, key), child, rows)
		}
	case []interface{}:
		*rows = append(*rows, permissionRow{Path: strings.Join(path, "."), Value: compactJSON(v)})
	default:
		*rows = append(*rows, permissionRow{Path: strings.Join(path, "."), Value: fmt.Sprint(v)})
	}
}

func compactJSON(value interface{}) string {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(b)
}
