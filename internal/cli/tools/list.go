package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all tools",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		allTools, err := client.ListTools(ctx)
		if err != nil {
			return err
		}

		filterFlag, _ := cmd.Flags().GetString("filter")
		var tools []api.Tool
		switch filterFlag {
		case "":
			tools = allTools
		case "public":
			for _, t := range allTools {
				if toolIsPublic(t) {
					tools = append(tools, t)
				}
			}
		case "private":
			for _, t := range allTools {
				if !toolIsPublic(t) {
					tools = append(tools, t)
				}
			}
		default:
			return fmt.Errorf("invalid filter %q: must be 'public' or 'private'", filterFlag)
		}

		sort.Slice(tools, func(i, j int) bool {
			return tools[i].Name < tools[j].Name
		})

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			b, err := json.MarshalIndent(tools, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		if len(tools) == 0 {
			if filterFlag != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "No tools found matching filter %q.\n", filterFlag)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No tools found.")
			}
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tID\tVISIBILITY\tGRANTS\tUPDATED")
		for _, t := range tools {
			updated := "-"
			if t.UpdatedAt > 0 {
				updated = time.Unix(t.UpdatedAt, 0).Format("2006-01-02")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				t.Name, t.ID, toolVisibility(t), len(t.AccessGrants), updated,
			)
		}
		w.Flush()

		if filterFlag != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d tool(s) matching filter %q.\n", len(tools), filterFlag)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d tool(s).\n", len(tools))
		}
		return nil
	},
}
