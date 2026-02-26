package root

import (
	"fmt"
	"os"

	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/cobra"
)

var (
	instance string
	output   string
	filter   string

	// Cmd represents the base command when called without any subcommands
	Cmd = &cobra.Command{
		Use:   "owui",
		Short: "owui is a CLI to manage Open WebUI instances",
		Long:  `A fast and flexible CLI written in Go for managing multiple Open WebUI instances.`,
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
	cobra.OnInitialize(initConfig)

	Cmd.PersistentFlags().StringVarP(&instance, "instance", "i", "", "instance name to use (default: active_instance from config)")
	Cmd.PersistentFlags().StringVarP(&output, "output", "o", "", "output format (console or json)")
	Cmd.PersistentFlags().StringVarP(&filter, "filter", "f", "", "filter results")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	_, err := config.Load()
	if err != nil {
		// Just print error, we can still run help commands
		fmt.Printf("Error loading configuration: %v\n", err)
	}
}
