package root

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/christestet/owui-go/internal/cli/version"
	"github.com/christestet/owui-go/internal/config"
	"github.com/christestet/owui-go/internal/updater"
	"github.com/spf13/cobra"
)

var (
	instance string
	output   string
	filter   string

	// validOutputFormats defines the allowed values for --output flag
	validOutputFormats = map[string]bool{
		"":       true,
		"pretty": true,
		"json":   true,
	}

	// Cmd represents the base command when called without any subcommands
	Cmd = &cobra.Command{
		Use:   "owui",
		Short: "owui is a CLI to manage Open WebUI instances",
		Long:  `A fast and flexible CLI written in Go for managing multiple Open WebUI instances.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !validOutputFormats[output] {
				return fmt.Errorf("invalid output format %q: must be 'pretty' or 'json'", output)
			}
			return nil
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := Cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfigCommand)

	Cmd.PersistentFlags().StringVarP(&instance, "instance", "i", "", "instance name to use (default: active_instance from config)")
	Cmd.PersistentFlags().StringVarP(&output, "output", "o", "", "output format (pretty or json)")
	Cmd.PersistentFlags().StringVarP(&filter, "filter", "f", "", "filter results")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	_, err := config.Load()
	if err != nil {
		// Just print error, we can still run help commands
		fmt.Printf("Error loading configuration: %v\n", err)
	}
	return err
}

func initConfigCommand() {
	_ = initConfig()
	go backgroundUpdateCheck()
}

func backgroundUpdateCheck() {
	if version.Version == "dev" {
		return
	}
	cfg, err := config.Load()
	if err != nil {
		return
	}
	if !updater.ShouldCheck(cfg.Cli.LastUpdateCheck) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	release, updateAvailable, err := updater.CheckLatest(ctx, version.Version)
	if err != nil {
		return
	}
	cfg.Cli.LastUpdateCheck = time.Now().UTC().Format(time.RFC3339)
	_ = cfg.Save()

	if updateAvailable {
		fmt.Fprintf(os.Stderr,
			"\nA new version of owui is available: %s (you have %s)\nRun 'owui update' to upgrade.\n\n",
			release.Version(), version.Version,
		)
	}
}
