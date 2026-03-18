package instances

import (
	"encoding/json"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the health of all instances or a specified instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		targetInstance, _ := cmd.Flags().GetString("instance")
		outputFormat, _ := cmd.Flags().GetString("output")

		ctx := shared.CmdContext(cmd)

		type HealthStatus struct {
			Name   string `json:"name"`
			URL    string `json:"url"`
			Status string `json:"status"`
			Active bool   `json:"active"`
			Error  string `json:"error,omitempty"`
		}

		checkInstance := func(name string, inst config.InstanceConfig) HealthStatus {
			client := api.NewClient(inst.URL, inst.APIKey, cfg.Settings.TimeoutSeconds)
			err := client.Healthcheck(ctx)
			status := "HEALTHY"
			hs := HealthStatus{
				Name:   name,
				URL:    inst.URL,
				Active: name == cfg.ActiveInstance,
			}
			if err != nil {
				status = "DOWN"
				hs.Error = err.Error()
			}
			hs.Status = status
			return hs
		}

		// Single instance mode
		if targetInstance != "" {
			inst, ok := cfg.Instances[targetInstance]
			if !ok {
				return fmt.Errorf("instance %q not found in config", targetInstance)
			}
			hs := checkInstance(targetInstance, inst)

			if outputFormat == "json" {
				b, marshalErr := json.MarshalIndent(hs, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				if hs.Error != "" {
					return fmt.Errorf("instance %q is not healthy", targetInstance)
				}
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tURL\tSTATUS\tACTIVE")
			activeMark := ""
			if hs.Active {
				activeMark = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", hs.Name, hs.URL, hs.Status, activeMark)
			w.Flush()

			if hs.Error != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "\nError details: %v\n", hs.Error)
				return fmt.Errorf("instance %q is not healthy", targetInstance)
			}
			return nil
		}

		// All instances mode
		if len(cfg.Instances) == 0 {
			return fmt.Errorf("no instances configured")
		}

		names := make([]string, 0, len(cfg.Instances))
		for name := range cfg.Instances {
			names = append(names, name)
		}
		sort.Strings(names)

		var statuses []HealthStatus
		for _, name := range names {
			statuses = append(statuses, checkInstance(name, cfg.Instances[name]))
		}

		if outputFormat == "json" {
			b, marshalErr := json.MarshalIndent(statuses, "", "  ")
			if marshalErr != nil {
				return marshalErr
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			var unhealthy []string
			for _, s := range statuses {
				if s.Error != "" {
					unhealthy = append(unhealthy, s.Name)
				}
			}
			if len(unhealthy) > 0 {
				return fmt.Errorf("unhealthy instances: %v", unhealthy)
			}
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tURL\tSTATUS\tACTIVE")
		var unhealthy []string
		for _, hs := range statuses {
			activeMark := ""
			if hs.Active {
				activeMark = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", hs.Name, hs.URL, hs.Status, activeMark)
			if hs.Error != "" {
				unhealthy = append(unhealthy, hs.Name)
			}
		}
		w.Flush()

		if len(unhealthy) > 0 {
			for _, hs := range statuses {
				if hs.Error != "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "\nError [%s]: %v\n", hs.Name, hs.Error)
				}
			}
			return fmt.Errorf("unhealthy instances: %v", unhealthy)
		}
		return nil
	},
}
