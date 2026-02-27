package groups

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:               "update [group]",
	Short:             "Update a group's name, description, or permissions",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: localGroupCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := resolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		groups, err := client.ListGroups(ctx)
		if err != nil {
			return err
		}
		localGroups := filterLocalGroups(groups)

		if len(localGroups) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No local groups found.")
			return nil
		}

		var groupName string
		if len(args) > 0 {
			groupName = args[0]
		} else {
			options := make([]huh.Option[string], 0, len(localGroups))
			for _, g := range localGroups {
				options = append(options, huh.NewOption(g.Name, g.Name))
			}
			err := runSearchableSelect("Select group to update", options, &groupName)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
		}

		group, err := findGroupByName(localGroups, groupName)
		if err != nil {
			return err
		}

		// Fetch full group details
		fullGroup, err := client.GetGroup(ctx, group.ID)
		if err != nil {
			return err
		}

		newName, _ := cmd.Flags().GetString("name")
		newDescription, _ := cmd.Flags().GetString("description")
		permissionsStr, _ := cmd.Flags().GetString("permissions")

		// Interactive mode if no flags provided
		flagsProvided := cmd.Flags().Changed("name") || cmd.Flags().Changed("description") || cmd.Flags().Changed("permissions")
		if !flagsProvided {
			if err := runUpdateWizard(fullGroup, &newName, &newDescription, &permissionsStr); err != nil {
				return wrapInteractiveCancelled(err)
			}
		}

		// Use existing values for unchanged fields
		if newName == "" {
			newName = fullGroup.Name
		}
		if newDescription == "" {
			newDescription = fullGroup.Description
		}

		var permissions json.RawMessage
		if permissionsStr != "" {
			if !json.Valid([]byte(permissionsStr)) {
				return fmt.Errorf("invalid JSON for permissions: %s", permissionsStr)
			}
			permissions = json.RawMessage(permissionsStr)
		} else {
			permissions = fullGroup.Permissions
		}

		form := api.GroupUpdateForm{
			Name:        newName,
			Description: newDescription,
			Permissions: permissions,
		}

		var confirmed bool
		err = huh.NewConfirm().
			Title(fmt.Sprintf("Confirm updating group '%s'?", group.Name)).
			Value(&confirmed).
			Run()
		if err != nil {
			return wrapInteractiveCancelled(err)
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		if _, err := client.UpdateGroup(ctx, group.ID, form); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated group '%s'\n", group.Name)
		return nil
	},
}

func init() {
	updateCmd.Flags().String("name", "", "new group name")
	updateCmd.Flags().String("description", "", "new group description")
	updateCmd.Flags().String("permissions", "", "new group permissions as JSON string")
}

func runUpdateWizard(group *api.Group, name, description, permissions *string) error {
	*name = group.Name
	*description = group.Description

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("New name (current: '%s')", group.Name)).
				Value(name),
			huh.NewInput().
				Title(fmt.Sprintf("New description (current: '%s')", group.Description)).
				Value(description),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	var setPermissions bool
	if err := huh.NewConfirm().Title("Update permissions?").Value(&setPermissions).Run(); err != nil {
		return err
	}

	if setPermissions {
		currentPerms := ""
		if len(group.Permissions) > 0 {
			currentPerms = string(group.Permissions)
		}
		var permInput string
		if currentPerms != "" && currentPerms != "null" {
			permInput = currentPerms
		}
		if err := huh.NewText().
			Title("Permissions (JSON)").
			Description("Enter permissions as a JSON object").
			Value(&permInput).
			Validate(func(s string) error {
				s = strings.TrimSpace(s)
				if s == "" {
					return nil
				}
				if !json.Valid([]byte(s)) {
					return fmt.Errorf("invalid JSON")
				}
				return nil
			}).
			Run(); err != nil {
			return err
		}
		*permissions = strings.TrimSpace(permInput)
	}

	return nil
}
