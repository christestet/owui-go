package pipelines

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:               "show [pipe_id]",
	Short:             "Show details for a pipe",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: pipeCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		inv, err := buildInventory(ctx, client, nil)
		if err != nil {
			return err
		}
		if len(inv.Pipes) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No pipes found.")
			return err
		}

		var pipeID string
		var explicitIdx *int
		if cmd.Flags().Changed("url-idx") {
			idx, _ := cmd.Flags().GetInt("url-idx")
			explicitIdx = &idx
		}

		if len(args) > 0 {
			pipeID = args[0]
		} else {
			options := make([]huh.Option[string], 0, len(inv.Pipes))
			for _, p := range inv.Pipes {
				label := fmt.Sprintf("%s (%s / idx=%d)", p.PipeID, p.RegistrationID, p.URLIdx)
				options = append(options, huh.NewOption(label, fmt.Sprintf("%s|%d", p.PipeID, p.URLIdx)))
			}
			var idx *int
			pipeID, idx, err = selectIDWithURLIdx("Select pipe", options, "invalid selected pipe value", "invalid selected pipe urlIdx")
			if err != nil {
				return err
			}
			explicitIdx = idx
		}

		pipe, err := resolvePipe(inv, pipeID, explicitIdx)
		if err != nil {
			return err
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			b, err := json.MarshalIndent(pipe, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return err
		}

		out := cmd.OutOrStdout()
		if _, err := fmt.Fprintf(out, "Pipe ID: %s\n", pipe.PipeID); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "Name: %s\n", pipe.Name); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "Description: %s\n", pipe.Description); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "Registration ID: %s\n", pipe.RegistrationID); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "Registration URL: %s\n", pipe.RegistrationURL); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "URL IDX: %d\n", pipe.URLIdx); err != nil {
			return err
		}

		raw, err := json.MarshalIndent(pipe.Raw, "", "  ")
		if err == nil {
			if _, err := fmt.Fprintln(out, "Raw:"); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out, string(raw)); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	showCmd.Flags().Int("url-idx", 0, "override urlIdx when resolving duplicate pipe IDs")
}
