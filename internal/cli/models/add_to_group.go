package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var addToGroupCmd = &cobra.Command{
	Use:   "add-to-group",
	Short: "Add model(s) to group(s)",
	Long: `Add one or more models to one or more groups, granting group members access.

Two modes:
  --model <name> --groups <g1> [g2 ...]    (one model to multiple groups)
  --models <m1> [m2 ...] --group <name>    (multiple models to one group)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		allModels, err := client.ListModels(ctx)
		if err != nil {
			return err
		}

		allGroups, err := client.ListGroups(ctx)
		if err != nil {
			return err
		}
		localGroups := shared.FilterLocalGroups(allGroups)

		if len(localGroups) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No local groups found.")
			return nil
		}

		modelFlag, _ := cmd.Flags().GetString("model")
		modelsFlag, _ := cmd.Flags().GetStringSlice("models")
		groupFlag, _ := cmd.Flags().GetString("group")
		groupsFlag, _ := cmd.Flags().GetStringSlice("groups")

		// Determine mode
		hasModelFlag := modelFlag != "" || len(modelsFlag) > 0
		hasGroupFlag := groupFlag != "" || len(groupsFlag) > 0

		var modelIDs []string
		var groupNames []string

		if !hasModelFlag && !hasGroupFlag {
			// Interactive wizard
			var mode string
			modeOptions := []huh.Option[string]{
				huh.NewOption("Add one model to multiple groups", "one-to-many"),
				huh.NewOption("Add multiple models to one group", "many-to-one"),
			}
			err := prompts.RunSearchableSelect("Select mode", modeOptions, &mode)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}

			if mode == "one-to-many" {
				// Select one model
				modelOptions := make([]huh.Option[string], 0, len(allModels))
				for _, m := range allModels {
					modelOptions = append(modelOptions, huh.NewOption(fmt.Sprintf("%s (%s)", m.Name, m.ID), m.ID))
				}
				var selectedModel string
				err := prompts.RunSearchableSelect("Select model", modelOptions, &selectedModel)
				if err != nil {
					return prompts.WrapInteractiveCancelled(err)
				}
				modelIDs = []string{selectedModel}

				// Select groups
				groupOptions := make([]huh.Option[string], 0, len(localGroups))
				for _, g := range localGroups {
					groupOptions = append(groupOptions, huh.NewOption(g.Name, g.Name))
				}
				err = prompts.RunSearchableMultiSelect("Select groups to grant access", groupOptions, &groupNames)
				if err != nil {
					return prompts.WrapInteractiveCancelled(err)
				}
			} else {
				// Select multiple models
				modelOptions := make([]huh.Option[string], 0, len(allModels))
				for _, m := range allModels {
					modelOptions = append(modelOptions, huh.NewOption(fmt.Sprintf("%s (%s)", m.Name, m.ID), m.ID))
				}
				err := prompts.RunSearchableMultiSelect("Select models to add", modelOptions, &modelIDs)
				if err != nil {
					return prompts.WrapInteractiveCancelled(err)
				}

				// Select one group
				groupOptions := make([]huh.Option[string], 0, len(localGroups))
				for _, g := range localGroups {
					groupOptions = append(groupOptions, huh.NewOption(g.Name, g.Name))
				}
				var selectedGroup string
				err = prompts.RunSearchableSelect("Select group", groupOptions, &selectedGroup)
				if err != nil {
					return prompts.WrapInteractiveCancelled(err)
				}
				groupNames = []string{selectedGroup}
			}
		} else {
			// Non-interactive
			if modelFlag != "" {
				modelIDs = []string{modelFlag}
			} else {
				modelIDs = modelsFlag
			}
			if groupFlag != "" {
				groupNames = []string{groupFlag}
			} else {
				groupNames = groupsFlag
			}
		}

		if len(modelIDs) == 0 || len(groupNames) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No models or groups selected.")
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

		// Resolve groups
		var resolvedGroups []api.Group
		for _, name := range groupNames {
			g, err := shared.FindGroupByName(localGroups, name)
			if err != nil {
				return err
			}
			resolvedGroups = append(resolvedGroups, *g)
		}

		// Confirmation
		modelNames := make([]string, 0, len(resolvedModels))
		for _, m := range resolvedModels {
			modelNames = append(modelNames, m.Name)
		}
		resolvedGroupNames := make([]string, 0, len(resolvedGroups))
		for _, g := range resolvedGroups {
			resolvedGroupNames = append(resolvedGroupNames, g.Name)
		}

		var confirmMsg string
		if len(resolvedModels) == 1 {
			confirmMsg = fmt.Sprintf("Confirm adding model '%s' to %d group(s): %s?", resolvedModels[0].Name, len(resolvedGroups), strings.Join(resolvedGroupNames, ", "))
		} else {
			confirmMsg = fmt.Sprintf("Confirm adding %d model(s) to group '%s': %s?", len(resolvedModels), resolvedGroups[0].Name, strings.Join(modelNames, ", "))
		}

		confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), confirmMsg)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		// Apply changes
		for _, m := range resolvedModels {
			// Fetch current model state
			currentModel, err := client.GetModel(ctx, m.ID)
			if err != nil {
				return fmt.Errorf("failed to fetch model '%s': %w", m.Name, err)
			}

			newGrants := make([]api.AccessGrantModel, len(currentModel.AccessGrants))
			copy(newGrants, currentModel.AccessGrants)

			for _, g := range resolvedGroups {
				// Check if already exists
				alreadyGranted := false
				for _, existing := range newGrants {
					if existing.PrincipalID == g.ID && existing.PrincipalType == "group" {
						alreadyGranted = true
						break
					}
				}
				if alreadyGranted {
					fmt.Fprintf(cmd.OutOrStdout(), "Warning: Model '%s' already has access grant for group '%s', skipping.\n", m.Name, g.Name)
					continue
				}

				newGrants = append(newGrants, api.AccessGrantModel{
					PrincipalType: "group",
					PrincipalID:   g.ID,
					Permission:    "read",
				})
			}

			form := api.ModelAccessGrantsForm{
				ID:           m.ID,
				Name:         m.Name,
				AccessGrants: newGrants,
			}
			if err := client.UpdateModelAccess(ctx, form); err != nil {
				return fmt.Errorf("failed to update access for model '%s': %w", m.Name, err)
			}

			for _, g := range resolvedGroups {
				fmt.Fprintf(cmd.OutOrStdout(), "Successfully added model '%s' to group '%s'\n", m.Name, g.Name)
			}
		}

		return nil
	},
}

func init() {
	addToGroupCmd.Flags().String("model", "", "single model name/id to add to groups")
	addToGroupCmd.Flags().StringSlice("models", nil, "model names/ids to add to a group")
	addToGroupCmd.Flags().String("group", "", "single group name to add models to")
	addToGroupCmd.Flags().StringSlice("groups", nil, "group names to add a model to")

	_ = addToGroupCmd.RegisterFlagCompletionFunc("model", modelCompletionFunc)
	_ = addToGroupCmd.RegisterFlagCompletionFunc("models", multiModelCompletionFunc)
	_ = addToGroupCmd.RegisterFlagCompletionFunc("group", localGroupCompletionFunc)
	_ = addToGroupCmd.RegisterFlagCompletionFunc("groups", localGroupCompletionFunc)
}
