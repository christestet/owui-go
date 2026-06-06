package pipelines

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

func resolveTargetPipe(cmd *cobra.Command, args []string) (*apiPipeTarget, error) {
	client, err := shared.ResolveClient(cmd)
	if err != nil {
		return nil, err
	}
	ctx := shared.CmdContext(cmd)
	inv, err := buildInventory(ctx, client, nil)
	if err != nil {
		return nil, err
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
		if len(inv.Pipes) == 0 {
			return nil, fmt.Errorf("no pipes found")
		}
		selected := ""
		options := make([]huh.Option[string], 0, len(inv.Pipes))
		for _, p := range inv.Pipes {
			label := fmt.Sprintf("%s (%s / idx=%d)", p.PipeID, p.RegistrationID, p.URLIdx)
			options = append(options, huh.NewOption(label, fmt.Sprintf("%s|%d", p.PipeID, p.URLIdx)))
		}
		err := prompts.RunSearchableSelect("Select pipe", options, &selected)
		if err != nil {
			return nil, prompts.WrapInteractiveCancelled(err)
		}
		parts := strings.SplitN(selected, "|", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid selected pipe value: %q", selected)
		}
		pipeID = parts[0]
		idx, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid selected pipe urlIdx: %w", err)
		}
		explicitIdx = &idx
	}

	pipe, err := resolvePipe(inv, pipeID, explicitIdx)
	if err != nil {
		return nil, err
	}
	return &apiPipeTarget{client: client, ctx: ctx, pipe: *pipe}, nil
}

type apiPipeTarget struct {
	client *api.Client
	ctx    context.Context
	pipe   Pipe
}

var valvesShowCmd = &cobra.Command{
	Use:               "show [pipe_id]",
	Short:             "Show current valves for a pipe",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: pipeCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := resolveTargetPipe(cmd, args)
		if err != nil {
			return err
		}
		resp, err := target.client.GetPipelineValvesRaw(target.ctx, target.pipe.PipeID, target.pipe.URLIdx)
		if err != nil {
			return err
		}
		return renderUntyped(cmd, resp)
	},
}

var valvesSpecCmd = &cobra.Command{
	Use:               "spec [pipe_id]",
	Short:             "Show valves spec for a pipe",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: pipeCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := resolveTargetPipe(cmd, args)
		if err != nil {
			return err
		}
		resp, err := target.client.GetPipelineValvesSpecRaw(target.ctx, target.pipe.PipeID, target.pipe.URLIdx)
		if err != nil {
			return err
		}
		return renderUntyped(cmd, resp)
	},
}

var valvesUpdateCmd = &cobra.Command{
	Use:               "update [pipe_id]",
	Short:             "Update valves for a pipe",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: pipeCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := resolveTargetPipe(cmd, args)
		if err != nil {
			return err
		}

		dataText, _ := cmd.Flags().GetString("data")
		if dataText == "" {
			err := huh.NewText().
				Title("Enter valves JSON").
				Value(&dataText).
				Run()
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(dataText), &payload); err != nil {
			return fmt.Errorf("invalid --data JSON: %w", err)
		}
		if payload == nil {
			return fmt.Errorf("invalid --data JSON: expected object")
		}

		confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Confirm updating valves for pipe '%s' (urlIdx=%d)?", target.pipe.PipeID, target.pipe.URLIdx))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		resp, err := target.client.UpdatePipelineValvesRaw(target.ctx, target.pipe.PipeID, target.pipe.URLIdx, payload)
		if err != nil {
			return err
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			b, err := json.MarshalIndent(map[string]any{
				"pipe_id": target.pipe.PipeID,
				"url_idx": target.pipe.URLIdx,
				"result":  resp,
			}, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated valves for pipe '%s'\n", target.pipe.PipeID)
		return nil
	},
}

func renderUntyped(cmd *cobra.Command, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}

func init() {
	valvesShowCmd.Flags().Int("url-idx", 0, "override urlIdx when resolving duplicate pipe IDs")
	valvesSpecCmd.Flags().Int("url-idx", 0, "override urlIdx when resolving duplicate pipe IDs")
	valvesUpdateCmd.Flags().Int("url-idx", 0, "override urlIdx when resolving duplicate pipe IDs")
	valvesUpdateCmd.Flags().String("data", "", "valves JSON object payload")
}
