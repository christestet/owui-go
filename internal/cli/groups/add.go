package groups

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new group",
	Long:  `Create a new group in Open WebUI. Provide flags for non-interactive mode, or omit them to use the interactive wizard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		permissionsStr, _ := cmd.Flags().GetString("permissions")
		usersFlag, _ := cmd.Flags().GetStringSlice("users")

		// Interactive mode if required fields are missing
		if name == "" || description == "" {
			if err := runAddWizard(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), client, &name, &description, &permissionsStr, &usersFlag); err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		if name == "" {
			return fmt.Errorf("name is required")
		}
		if description == "" {
			return fmt.Errorf("description is required")
		}

		// Parse permissions
		var permissions json.RawMessage
		if permissionsStr != "" {
			if !json.Valid([]byte(permissionsStr)) {
				return fmt.Errorf("invalid JSON for permissions: %s", permissionsStr)
			}
			permissions = json.RawMessage(permissionsStr)
		}

		form := api.GroupForm{
			Name:        name,
			Description: description,
			Permissions: permissions,
		}

		group, err := client.CreateGroup(ctx, form)
		if err != nil {
			return err
		}

		// Add users if specified
		if len(usersFlag) > 0 {
			allUsers, err := client.ListUsers(ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch users: %w", err)
			}

			resolvedUsers, err := shared.ResolveUsers(allUsers, usersFlag)
			if err != nil {
				return err
			}
			userIDs := make([]string, 0, len(resolvedUsers))
			for _, u := range resolvedUsers {
				if u.Role != "user" {
					return fmt.Errorf("user %q has role %q; only users with role 'user' can be added to groups", u.Name, u.Role)
				}
				userIDs = append(userIDs, u.ID)
			}

			if err := client.AddUsersToGroup(ctx, group.ID, userIDs); err != nil {
				return fmt.Errorf("group created but failed to add users: %w", err)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Successfully created group '%s' with %d member(s)\n", group.Name, len(userIDs))
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Successfully created group '%s'\n", group.Name)
		return err
	},
}

func init() {
	addCmd.Flags().String("name", "", "group name")
	addCmd.Flags().String("description", "", "group description")
	addCmd.Flags().String("permissions", "", "group permissions as JSON string")
	addCmd.Flags().StringSlice("users", nil, "user IDs, emails, usernames, or unique names to add (only role 'user')")
}

func runAddWizard(ctx context.Context, in io.Reader, out io.Writer, client *api.Client, name, description, permissions *string, users *[]string) error {
	// Basic fields
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Group name").
				Value(name),
			huh.NewInput().
				Title("Description").
				Value(description),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	// Ask about adding users
	addUsers, err := prompts.ConfirmYN(in, out, "Add users to this group?")
	if err != nil {
		return err
	}

	if addUsers {
		allUsers, err := client.ListUsers(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch users: %w", err)
		}

		eligibleUsers := shared.FilterUsersByRole(allUsers, "user")
		if len(eligibleUsers) == 0 {
			if _, err := fmt.Fprintln(out, "No eligible users found (only users with role 'user' can be added)."); err != nil {
				return err
			}
		} else {
			var selectedIDs []string
			if err := prompts.RunSearchableMultiSelect("Select users to add", shared.UserOptions(eligibleUsers), &selectedIDs); err != nil {
				return err
			}
			*users = selectedIDs
		}
	}

	// Ask about permissions
	setPermissions, err := prompts.ConfirmYN(in, out, "Set custom permissions?")
	if err != nil {
		return err
	}

	if setPermissions {
		var permInput string
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
