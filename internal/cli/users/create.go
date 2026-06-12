package users

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	Long:  `Create a new user in Open WebUI. Provide flags for non-interactive mode, or omit them to use the interactive wizard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")
		role, _ := cmd.Flags().GetString("role")

		// If any required field is missing, run interactive mode
		if name == "" || email == "" || password == "" || role == "" {
			if err := runCreateWizard(&name, &email, &password, &role); err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		if name == "" {
			return fmt.Errorf("name is required")
		}
		if email == "" {
			return fmt.Errorf("email is required")
		}
		if password == "" {
			return fmt.Errorf("password is required")
		}

		ctx := shared.CmdContext(cmd)

		form := api.CreateUserForm{
			Name:     name,
			Email:    email,
			Password: password,
			Role:     role,
		}

		if err := client.CreateUser(ctx, form); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Successfully created user %s with role %s\n", name, role)
		return err
	},
}

func init() {
	createCmd.Flags().String("name", "", "user display name")
	createCmd.Flags().String("email", "", "user email address")
	createCmd.Flags().String("password", "", "user password")
	createCmd.Flags().String("role", "", "user role (admin, user, pending)")
	_ = createCmd.RegisterFlagCompletionFunc("role", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validRoles, cobra.ShellCompDirectiveNoFileComp
	})
}

func runCreateWizard(name, email, password, role *string) error {
	if *role == "" {
		*role = "pending"
	}

	roleOptions := []huh.Option[string]{
		huh.NewOption("admin", "admin"),
		huh.NewOption("user", "user"),
		huh.NewOption("pending", "pending"),
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Description("Display name for the user").
				Value(name),
			huh.NewInput().
				Title("Email").
				Description("Email address for the user").
				Value(email),
			huh.NewInput().
				Title("Password").
				Description("Password for the user").
				EchoMode(huh.EchoModePassword).
				Value(password),
			prompts.RunSearchableSelectWithDescription("Select role", "Role for the user", roleOptions, role),
		),
	)
	return form.Run()
}
