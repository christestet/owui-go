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

var removeFromGroupCmd = &cobra.Command{
	Use:   "remove-from-group [model_id]",
	Short: "Remove a model from group(s)",
	Long:  `Remove a model from one or more groups, revoking group members' access.`,
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: smartModelCompletionFunc(func(m api.ModelAccessResponse) bool {
		return isPrivate(m) // Only show models with access grants
	}),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Filter to only private models (those with access grants)
		var privateModels []api.ModelAccessResponse
		for _, m := range allModels {
			if isPrivate(m) {
				privateModels = append(privateModels, m)
			}
		}

		// Fetch all groups for name resolution
		allGroups, err := client.ListGroups(ctx)
		if err != nil {
			return err
		}
		groupMap := make(map[string]string) // ID -> Name
		for _, g := range allGroups {
			groupMap[g.ID] = g.Name
		}

		var modelID string
		if len(args) > 0 {
			modelID = args[0]
		} else {
			if len(privateModels) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No models with access grants found.")
				return nil
			}
			options := make([]huh.Option[string], 0, len(privateModels))
			for _, m := range privateModels {
				label := fmt.Sprintf("%s (%s) [%d groups]", m.Name, m.ID, len(m.AccessGrants))
				options = append(options, huh.NewOption(label, m.ID))
			}
			err := runSearchableSelect("Select model", options, &modelID)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
		}

		// Get current model state
		model, err := client.GetModel(ctx, modelID)
		if err != nil {
			return err
		}

		if len(model.AccessGrants) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Model '%s' has no access grants to remove.\n", model.Name)
			return nil
		}

		// Get assigned group names
		type assignedGroup struct {
			id   string
			name string
		}
		var assigned []assignedGroup
		for _, grant := range model.AccessGrants {
			if grant.PrincipalType == "group" {
				name := grant.PrincipalID
				if n, ok := groupMap[grant.PrincipalID]; ok {
					name = n
				}
				assigned = append(assigned, assignedGroup{id: grant.PrincipalID, name: name})
			}
		}

		groupsFlag, _ := cmd.Flags().GetStringSlice("groups")
		var selectedGroupNames []string
		if len(groupsFlag) > 0 {
			selectedGroupNames = groupsFlag
		} else {
			// Interactive multi-select
			options := make([]huh.Option[string], 0, len(assigned))
			for _, a := range assigned {
				options = append(options, huh.NewOption(a.name, a.name))
			}
			err := runSearchableMultiSelect("Select groups to revoke access", options, &selectedGroupNames)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
		}

		if len(selectedGroupNames) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No groups selected.")
			return nil
		}

		// Resolve selected group names to IDs
		removeGroupIDs := make(map[string]bool)
		for _, name := range selectedGroupNames {
			found := false
			for _, a := range assigned {
				if a.name == name {
					removeGroupIDs[a.id] = true
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("group %q is not assigned to model '%s'", name, model.Name)
			}
		}

		confirmed, err := prompts.ConfirmYN(fmt.Sprintf("Confirm removing model '%s' from %d group(s): %s?", model.Name, len(selectedGroupNames), strings.Join(selectedGroupNames, ", ")))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		// Build remaining grants
		var remainingGrants []api.AccessGrantModel
		for _, grant := range model.AccessGrants {
			if grant.PrincipalType == "group" && removeGroupIDs[grant.PrincipalID] {
				continue
			}
			remainingGrants = append(remainingGrants, grant)
		}
		if remainingGrants == nil {
			remainingGrants = []api.AccessGrantModel{}
		}

		form := api.ModelAccessGrantsForm{
			ID:           model.ID,
			Name:         model.Name,
			AccessGrants: remainingGrants,
		}
		if err := client.UpdateModelAccess(ctx, form); err != nil {
			return fmt.Errorf("failed to update access for model '%s': %w", model.Name, err)
		}

		for _, name := range selectedGroupNames {
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully removed model '%s' from group '%s'\n", model.Name, name)
		}

		// Show remaining status
		remainingGroupGrants := 0
		var remainingGroupNames []string
		for _, grant := range remainingGrants {
			if grant.PrincipalType == "group" {
				remainingGroupGrants++
				name := grant.PrincipalID
				if n, ok := groupMap[grant.PrincipalID]; ok {
					name = n
				}
				remainingGroupNames = append(remainingGroupNames, name)
			}
		}

		if remainingGroupGrants == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Note: Model '%s' has no remaining group access -- it is now admin-only.\n", model.Name)
			fmt.Fprintln(cmd.OutOrStdout(), "Tip: Use 'owui models set-visibility public' to make it accessible to all users.")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Note: Model '%s' still has access grants for %d group(s): %s\n", model.Name, remainingGroupGrants, strings.Join(remainingGroupNames, ", "))
		}

		return nil
	},
}

func init() {
	removeFromGroupCmd.Flags().StringSlice("groups", nil, "group names to remove the model from")

	_ = removeFromGroupCmd.RegisterFlagCompletionFunc("groups", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		// Smart completion: only show groups the model is assigned to
		client := resolveClientForCompletion()
		if client == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		model, err := client.GetModel(context.Background(), args[0])
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		groups, err := client.ListGroups(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		groupMap := make(map[string]string)
		for _, g := range groups {
			groupMap[g.ID] = g.Name
		}
		var comps []string
		for _, grant := range model.AccessGrants {
			if grant.PrincipalType == "group" {
				name := grant.PrincipalID
				if n, ok := groupMap[grant.PrincipalID]; ok {
					name = n
				}
				if strings.HasPrefix(name, toComplete) {
					comps = append(comps, name)
				}
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	})
}
