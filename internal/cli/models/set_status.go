package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/spf13/cobra"
)

var setStatusCmd = &cobra.Command{
	Use:   "set-status <enable|disable> [model_id ...]",
	Short: "Enable or disable models",
	Long:  `Enable or disable one or more models by toggling their is_active state.`,
	Args:  cobra.MinimumNArgs(0),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Complete action: enable or disable
			var comps []string
			for _, action := range []string{"enable", "disable"} {
				if strings.HasPrefix(action, toComplete) {
					comps = append(comps, action)
				}
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		}
		// Smart completion: enable shows disabled models, disable shows enabled models
		action := args[0]
		var filterFn func(api.ModelAccessResponse) bool
		switch action {
		case "enable":
			filterFn = func(m api.ModelAccessResponse) bool { return !m.IsActive }
		case "disable":
			filterFn = func(m api.ModelAccessResponse) bool { return m.IsActive }
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return smartModelCompletionFunc(filterFn)(cmd, args[1:], toComplete)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("action required: 'enable' or 'disable'")
		}

		action := args[0]
		if action != "enable" && action != "disable" {
			return fmt.Errorf("invalid action %q: must be 'enable' or 'disable'", action)
		}

		client, err := resolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		allModels, err := client.ListModels(ctx)
		if err != nil {
			return err
		}

		// Filter models based on action
		var eligibleModels []api.ModelAccessResponse
		for _, m := range allModels {
			if action == "enable" && !m.IsActive {
				eligibleModels = append(eligibleModels, m)
			} else if action == "disable" && m.IsActive {
				eligibleModels = append(eligibleModels, m)
			}
		}

		modelIDs := args[1:]
		if len(modelIDs) == 0 {
			// Interactive multi-select
			if len(eligibleModels) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No models to %s.\n", action)
				return nil
			}
			options := make([]huh.Option[string], 0, len(eligibleModels))
			for _, m := range eligibleModels {
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", m.Name, m.ID), m.ID))
			}
			var selectedIDs []string
			err := runSearchableMultiSelect(fmt.Sprintf("Select models to %s", action), options, &selectedIDs)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
			modelIDs = selectedIDs
		}

		if len(modelIDs) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No models selected.")
			return nil
		}

		// Resolve models
		var resolvedModels []api.ModelAccessResponse
		for _, id := range modelIDs {
			m, err := findModelByNameOrID(allModels, id)
			if err != nil {
				return err
			}
			resolvedModels = append(resolvedModels, *m)
		}

		// Build name list for confirmation
		names := make([]string, 0, len(resolvedModels))
		for _, m := range resolvedModels {
			names = append(names, m.Name)
		}

		verb := "enabling"
		if action == "disable" {
			verb = "disabling"
		}
		confirmed, err := prompts.ConfirmYN(fmt.Sprintf("Confirm %s %d model(s): %s?", verb, len(resolvedModels), strings.Join(names, ", ")))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		for _, m := range resolvedModels {
			if _, err := client.ToggleModel(ctx, m.ID); err != nil {
				return fmt.Errorf("failed to %s model '%s': %w", action, m.Name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully %sd model '%s'\n", action, m.Name)
		}

		return nil
	},
}
