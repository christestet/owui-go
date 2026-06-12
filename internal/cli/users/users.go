package users

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

const msgNoUsersFound = "No users found."

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users in an Open WebUI instance",
}

// Register adds all user-related subcommands to the root command.
func Register(rootCmd *cobra.Command) {
	usersCmd.AddCommand(listCmd)
	usersCmd.AddCommand(createCmd)
	usersCmd.AddCommand(removeCmd)
	usersCmd.AddCommand(updateRoleCmd)
	usersCmd.AddCommand(addToGroupCmd)
	usersCmd.AddCommand(removeFromGroupCmd)

	rootCmd.AddCommand(usersCmd)
}

// userCompletionFunc returns a ValidArgsFunction that completes user names from the API.
func userCompletionFunc() func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		client := shared.ResolveClientForCompletion(cmd)
		if client == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		users, err := client.ListUsers(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var comps []string
		for _, u := range users {
			if strings.HasPrefix(u.Name, toComplete) {
				comps = append(comps, u.Name)
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	}
}

// multiUserCompletionFunc returns a ValidArgsFunction that allows completing multiple user names.
func multiUserCompletionFunc(roleFilter string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client := shared.ResolveClientForCompletion(cmd)
		if client == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		users, err := client.ListUsers(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		// Build set of already-selected names
		selected := make(map[string]bool)
		for _, a := range args {
			selected[a] = true
		}
		var comps []string
		for _, u := range users {
			if selected[u.Name] {
				continue
			}
			if roleFilter != "" && u.Role != roleFilter {
				continue
			}
			if strings.HasPrefix(u.Name, toComplete) {
				comps = append(comps, u.Name)
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	}
}

// --- list command ---

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all users",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		filterQuery, _ := cmd.Flags().GetString("filter")
		roleFilter, _ := cmd.Flags().GetString("role")

		var users []api.User
		if filterQuery != "" {
			users, err = client.ListUsersWithOptions(ctx, &api.UserListOptions{Query: filterQuery})
		} else {
			users, err = client.ListUsers(ctx)
		}
		if err != nil {
			return err
		}

		if roleFilter != "" {
			users = shared.FilterUsersByRole(users, roleFilter)
		}

		sort.Slice(users, func(i, j int) bool {
			return users[i].Name < users[j].Name
		})

		outputFormat, _ := cmd.Flags().GetString("output")

		if outputFormat == "json" {
			b, err := json.MarshalIndent(users, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return err
		}

		if len(users) == 0 {
			if filterQuery != "" {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "No users found matching %q.\n", filterQuery)
				return err
			} else {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), msgNoUsersFound)
				return err
			}
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		if _, err := fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE"); err != nil {
			return err
		}
		for _, u := range users {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", u.ID, u.Name, u.Email, u.Role); err != nil {
				return err
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}

		if filterQuery != "" {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d user(s) matching %q.\n", len(users), filterQuery)
			return err
		} else {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d user(s).\n", len(users))
			return err
		}
	},
}

// --- remove command ---

var removeCmd = &cobra.Command{
	Use:               "remove [username]",
	Short:             "Remove a user",
	Aliases:           []string{"rm"},
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: userCompletionFunc(),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		users, err := client.ListUsers(ctx)
		if err != nil {
			return err
		}

		var userName string
		if len(args) > 0 {
			userName = args[0]
		} else {
			// Interactive select
			options := make([]huh.Option[string], 0, len(users))
			for _, u := range users {
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", u.Name, u.Email), u.Name))
			}
			if len(options) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), msgNoUsersFound)
				return err
			}
			err := prompts.RunSearchableSelect("Select user to delete", options, &userName)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		user, err := shared.FindUserByName(users, userName)
		if err != nil {
			return err
		}

		confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Confirm deleting user %s?", user.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return err
		}

		if err := client.DeleteUser(ctx, user.ID); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted user %s\n", user.Name)
		return err
	},
}

// --- update-role command ---

var validRoles = []string{"admin", "user", "pending"}

var updateRoleCmd = &cobra.Command{
	Use:               "update-role [username]",
	Short:             "Update a user's role",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: userCompletionFunc(),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		users, err := client.ListUsers(ctx)
		if err != nil {
			return err
		}

		var userName string
		if len(args) > 0 {
			userName = args[0]
		} else {
			options := make([]huh.Option[string], 0, len(users))
			for _, u := range users {
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s) - %s", u.Name, u.Email, u.Role), u.Name))
			}
			if len(options) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), msgNoUsersFound)
				return err
			}
			err := prompts.RunSearchableSelect("Select user to update role", options, &userName)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		user, err := shared.FindUserByName(users, userName)
		if err != nil {
			return err
		}

		role, _ := cmd.Flags().GetString("role")
		if role == "" {
			roleOptions := make([]huh.Option[string], 0, len(validRoles))
			for _, r := range validRoles {
				roleOptions = append(roleOptions, huh.NewOption(r, r))
			}
			err := prompts.RunSearchableSelect("Select new role", roleOptions, &role)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Confirm updating user %s's role to %s?", user.Name, role))
		if err != nil {
			return err
		}
		if !confirmed {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return err
		}

		form := api.UpdateUserForm{Role: &role}
		if err := client.UpdateUser(ctx, user.ID, form); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated user %s's role to %s\n", user.Name, role)
		return err
	},
}

func init() {
	listCmd.Flags().String("role", "", "filter by role (admin, user, pending)")
	_ = listCmd.RegisterFlagCompletionFunc("role", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var comps []string
		for _, r := range validRoles {
			if strings.HasPrefix(r, toComplete) {
				comps = append(comps, r)
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	})

	updateRoleCmd.Flags().String("role", "", "new role (admin, user, pending)")
	_ = updateRoleCmd.RegisterFlagCompletionFunc("role", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var comps []string
		for _, r := range validRoles {
			if strings.HasPrefix(r, toComplete) {
				comps = append(comps, r)
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	})
}
