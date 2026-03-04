package pipelines

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:               "show [pipe_id]",
	Short:             "Show details for a pipe",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: pipeCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := resolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		inv, err := buildInventory(ctx, client, nil)
		if err != nil {
			return err
		}
		if len(inv.Pipes) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No pipes found.")
			return nil
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
			selected := ""
			options := make([]huh.Option[string], 0, len(inv.Pipes))
			for _, p := range inv.Pipes {
				label := fmt.Sprintf("%s (%s / idx=%d)", p.PipeID, p.RegistrationID, p.URLIdx)
				options = append(options, huh.NewOption(label, fmt.Sprintf("%s|%d", p.PipeID, p.URLIdx)))
			}
			err := runSearchableSelect("Select pipe", options, &selected)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
			parts := strings.SplitN(selected, "|", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid selected pipe value: %q", selected)
			}
			pipeID = parts[0]
			idx, err := strconv.Atoi(parts[1])
			if err != nil {
				return fmt.Errorf("invalid selected pipe urlIdx: %w", err)
			}
			explicitIdx = &idx
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
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Pipe ID: %s\n", pipe.PipeID)
		fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", pipe.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", pipe.Description)
		fmt.Fprintf(cmd.OutOrStdout(), "Registration ID: %s\n", pipe.RegistrationID)
		fmt.Fprintf(cmd.OutOrStdout(), "Registration URL: %s\n", pipe.RegistrationURL)
		fmt.Fprintf(cmd.OutOrStdout(), "URL IDX: %d\n", pipe.URLIdx)

		raw, err := json.MarshalIndent(pipe.Raw, "", "  ")
		if err == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Raw:")
			fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		}
		return nil
	},
}

func init() {
	showCmd.Flags().Int("url-idx", 0, "override urlIdx when resolving duplicate pipe IDs")
}
