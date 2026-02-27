package instances

import (
	"fmt"
	"net/url"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [instance-name]",
	Short: "Add a new instance",
	Long:  `Add a new Open WebUI instance. Provide flags for non-interactive mode, or omit them to use the interactive wizard.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		name := ""
		if len(args) > 0 {
			name = args[0]
		}

		instanceURL, _ := cmd.Flags().GetString("url")
		apiKey, _ := cmd.Flags().GetString("api-key")

		// If any required field is missing, run interactive mode
		if name == "" || instanceURL == "" || apiKey == "" {
			err := runAddWizard(&name, &instanceURL, &apiKey)
			if err != nil {
				return fmt.Errorf("interactive input cancelled: %w", err)
			}
		}

		if err := validateInstanceInput(name, instanceURL); err != nil {
			return err
		}

		if _, exists := cfg.Instances[name]; exists {
			return fmt.Errorf("instance %q already exists", name)
		}

		if cfg.Instances == nil {
			cfg.Instances = make(map[string]config.InstanceConfig)
		}

		cfg.Instances[name] = config.InstanceConfig{
			URL:     instanceURL,
			APIKey:  apiKey,
			AddedAt: time.Now().Format(time.RFC3339),
		}

		// If this is the first instance, set it as active
		if cfg.ActiveInstance == "" {
			cfg.ActiveInstance = name
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Instance %q added\n", name)
		if cfg.ActiveInstance == name {
			fmt.Fprintf(cmd.OutOrStdout(), "Set as active instance\n")
		}
		return nil
	},
}

func init() {
	addCmd.Flags().String("url", "", "instance URL (e.g. http://localhost:3000)")
	addCmd.Flags().String("api-key", "", "API key for authentication")
}

func validateInstanceInput(name, instanceURL string) error {
	if name == "" {
		return fmt.Errorf("instance name is required")
	}
	if instanceURL == "" {
		return fmt.Errorf("instance URL is required")
	}
	u, err := url.ParseRequestURI(instanceURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", instanceURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL %q: scheme must be http or https", instanceURL)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid URL %q: host is required", instanceURL)
	}
	return nil
}

func runAddWizard(name, instanceURL, apiKey *string) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Instance name").
				Description("A short name to identify this instance").
				Value(name),
			huh.NewInput().
				Title("URL").
				Description("The URL of the Open WebUI instance").
				Placeholder("http://localhost:3000").
				Value(instanceURL),
			huh.NewInput().
				Title("API Key").
				Description("The API key for authentication").
				EchoMode(huh.EchoModePassword).
				Value(apiKey),
		),
	)
	return form.Run()
}
