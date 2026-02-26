package instances

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the health of the active or specified instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// determine which instance to use
		targetInstance, _ := cmd.Flags().GetString("instance")
		if targetInstance == "" {
			targetInstance = cfg.ActiveInstance
		}

		if targetInstance == "" {
			return fmt.Errorf("no instance specified and no active instance configured")
		}

		inst, ok := cfg.Instances[targetInstance]
		if !ok {
			return fmt.Errorf("instance %q not found in config", targetInstance)
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		client := api.NewClient(inst.URL, inst.APIKey, cfg.Settings.TimeoutSeconds)
		err = client.Healthcheck()

		status := "HEALTHY"
		if err != nil {
			status = "DOWN"
		}

		if jsonOutput || outputFormat == "json" {
			type HealthStatus struct {
				Name   string `json:"name"`
				URL    string `json:"url"`
				Status string `json:"status"`
				Error  string `json:"error,omitempty"`
			}
			hs := HealthStatus{
				Name:   targetInstance,
				URL:    inst.URL,
				Status: status,
			}
			if err != nil {
				hs.Error = err.Error()
			}
			b, marshalErr := json.MarshalIndent(hs, "", "  ")
			if marshalErr != nil {
				return marshalErr
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tURL\tSTATUS")
		fmt.Fprintf(w, "%s\t%s\t%s\n", targetInstance, inst.URL, status)
		w.Flush()

		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "\nError details: %v\n", err)
		}

		return nil
	},
}
