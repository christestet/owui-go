package models

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
	Short:   "List all models",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		queryFlag, _ := cmd.Flags().GetString("query")
		tagFlag, _ := cmd.Flags().GetString("tag")

		var opts *api.ModelListOptions
		if queryFlag != "" || tagFlag != "" {
			opts = &api.ModelListOptions{Query: queryFlag, Tag: tagFlag}
		}

		allModels, err := client.ListModelsWithOptions(ctx, opts)
		if err != nil {
			return err
		}

		// Apply client-side filter
		filterType, _ := cmd.Flags().GetString("filter")
		var models []api.ModelAccessResponse
		switch filterType {
		case "enabled":
			for _, m := range allModels {
				if m.IsActive {
					models = append(models, m)
				}
			}
		case "disabled":
			for _, m := range allModels {
				if !m.IsActive {
					models = append(models, m)
				}
			}
		case "public":
			for _, m := range allModels {
				if isPublic(m) {
					models = append(models, m)
				}
			}
		case "private":
			for _, m := range allModels {
				if isPrivate(m) {
					models = append(models, m)
				}
			}
		case "":
			models = allModels
		default:
			return fmt.Errorf("invalid filter %q: must be 'enabled', 'disabled', 'public', or 'private'", filterType)
		}

		sort.Slice(models, func(i, j int) bool {
			return models[i].Name < models[j].Name
		})

		outputFormat, _ := cmd.Flags().GetString("output")

		if outputFormat == "json" {
			b, err := json.MarshalIndent(models, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return err
		}

		if len(models) == 0 {
			if filterType != "" {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "No models found matching filter %q.\n", filterType)
				return err
			} else {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No models found.")
				return err
			}
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		if _, err := fmt.Fprintln(w, "NAME\tID\tBASE MODEL\tSTATUS\tVISIBILITY\tGRANTS\tUPDATED"); err != nil {
			return err
		}
		for _, m := range models {
			updated := "-"
			if m.UpdatedAt > 0 {
				updated = time.Unix(m.UpdatedAt, 0).Format("2006-01-02")
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
				m.Name, m.ID, m.BaseModelID,
				modelStatus(m), modelVisibility(m),
				len(m.AccessGrants), updated,
			); err != nil {
				return err
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}

		if filterType != "" {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d model(s) matching filter %q.\n", len(models), filterType)
			return err
		} else {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d model(s).\n", len(models))
			return err
		}
	},
}

func init() {
	listCmd.Flags().String("query", "", "server-side search by model name/id")
	listCmd.Flags().String("tag", "", "filter models by tag")
}
