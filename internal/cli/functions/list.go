package functions

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
	Short:   "List all functions",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		allFunctions, err := client.ListFunctions(ctx)
		if err != nil {
			return err
		}

		typeFlag, _ := cmd.Flags().GetString("type")
		filterFlag, _ := cmd.Flags().GetString("filter")

		var functions []api.Function
		for _, f := range allFunctions {
			if typeFlag != "" && f.Type != typeFlag {
				continue
			}
			switch filterFlag {
			case "":
				// no filter
			case "enabled":
				if !f.IsActive {
					continue
				}
			case "disabled":
				if f.IsActive {
					continue
				}
			case "global":
				if !f.IsActive || !f.IsGlobal {
					continue
				}
			case "private":
				if !f.IsActive || f.IsGlobal {
					continue
				}
			default:
				return fmt.Errorf("invalid filter %q: must be 'enabled', 'disabled', 'global', or 'private'", filterFlag)
			}
			functions = append(functions, f)
		}

		sort.Slice(functions, func(i, j int) bool {
			return functions[i].Name < functions[j].Name
		})

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			b, err := json.MarshalIndent(functions, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return err
		}

		if len(functions) == 0 {
			if filterFlag != "" || typeFlag != "" {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No functions found matching the given filters.")
				return err
			} else {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No functions found.")
				return err
			}
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		if _, err := fmt.Fprintln(w, "NAME\tID\tTYPE\tSTATUS\tSCOPE\tUPDATED"); err != nil {
			return err
		}
		for _, f := range functions {
			updated := "-"
			if f.UpdatedAt > 0 {
				updated = time.Unix(f.UpdatedAt, 0).Format("2006-01-02")
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				f.Name, f.ID, f.Type,
				functionStatus(f), functionScope(f),
				updated,
			); err != nil {
				return err
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d function(s).\n", len(functions))
		return err
	},
}

func init() {
	listCmd.Flags().String("type", "", "filter by function type (e.g. filter, action, pipe)")
}
