package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/spf13/cobra"
)

var setVisibilityCmd = &cobra.Command{
	Use:   "set-visibility <public|private> [model_id ...]",
	Short: "Set model visibility to public or private",
	Long:  `Change the visibility of one or more models between public and private.`,
	Args:  cobra.MinimumNArgs(0),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			var comps []string
			for _, v := range []string{"public", "private"} {
				if strings.HasPrefix(v, toComplete) {
					comps = append(comps, v)
				}
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		}
		action := args[0]
		var filterFn func(api.ModelAccessResponse) bool
		switch action {
		case "public":
			filterFn = func(m api.ModelAccessResponse) bool { return isPrivate(m) }
		case "private":
			filterFn = func(m api.ModelAccessResponse) bool { return isPublic(m) }
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return smartModelCompletionFunc(filterFn)(cmd, args[1:], toComplete)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("visibility required: 'public' or 'private'")
		}

		visibility := args[0]
		if visibility != "public" && visibility != "private" {
			return fmt.Errorf("invalid visibility %q: must be 'public' or 'private'", visibility)
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

		// Filter models based on current visibility
		var eligibleModels []api.ModelAccessResponse
		for _, m := range allModels {
			if visibility == "public" && isPrivate(m) {
				eligibleModels = append(eligibleModels, m)
			} else if visibility == "private" && isPublic(m) {
				eligibleModels = append(eligibleModels, m)
			}
		}

		modelIDs := args[1:]
		if len(modelIDs) == 0 {
			if len(eligibleModels) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No models to make %s.\n", visibility)
				return nil
			}
			options := make([]huh.Option[string], 0, len(eligibleModels))
			for _, m := range eligibleModels {
				label := fmt.Sprintf("%s (%s)", m.Name, m.ID)
				if isPrivate(m) {
					label += fmt.Sprintf(" [%d grants]", len(m.AccessGrants))
				}
				options = append(options, huh.NewOption(label, m.ID))
			}
			var selectedIDs []string
			err := runSearchableMultiSelect(fmt.Sprintf("Select models to make %s", visibility), options, &selectedIDs)
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

		names := make([]string, 0, len(resolvedModels))
		for _, m := range resolvedModels {
			names = append(names, m.Name)
		}

		var confirmed bool
		err = huh.NewConfirm().
			Title(fmt.Sprintf("Confirm making %d model(s) %s: %s?", len(resolvedModels), visibility, strings.Join(names, ", "))).
			Value(&confirmed).
			Run()
		if err != nil {
			return wrapInteractiveCancelled(err)
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		for _, m := range resolvedModels {
			var grants []api.AccessGrantModel
			if visibility == "public" {
				// Clear all access grants
				grants = []api.AccessGrantModel{}
				if len(m.AccessGrants) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Note: This will remove all %d access grants from '%s'.\n", len(m.AccessGrants), m.Name)
				}
			} else {
				// Make private: clear all access grants (admin-only)
				grants = []api.AccessGrantModel{}
			}

			form := api.ModelAccessGrantsForm{
				ID:           m.ID,
				Name:         m.Name,
				AccessGrants: grants,
			}
			if err := client.UpdateModelAccess(ctx, form); err != nil {
				return fmt.Errorf("failed to set visibility for model '%s': %w", m.Name, err)
			}

			if visibility == "public" {
				fmt.Fprintf(cmd.OutOrStdout(), "Successfully set model '%s' to public\n", m.Name)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Successfully set model '%s' to private (admin-only access)\n", m.Name)
				fmt.Fprintln(cmd.OutOrStdout(), "Tip: Use 'owui models add-to-group' to grant access to specific groups.")
			}
		}

		return nil
	},
}
