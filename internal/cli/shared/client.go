package shared

import (
	"context"
	"fmt"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/cobra"
)

// targetInstanceName returns the instance to operate on: the --instance flag if
// set, otherwise the active instance from config.
func targetInstanceName(cmd *cobra.Command, cfg *config.Config) string {
	if cmd != nil {
		if name, _ := cmd.Flags().GetString("instance"); name != "" {
			return name
		}
	}
	return cfg.ActiveInstance
}

// ResolveClient loads config, resolves the target instance, and returns an API client.
func ResolveClient(cmd *cobra.Command) (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	targetInstance := targetInstanceName(cmd, cfg)
	if targetInstance == "" {
		return nil, fmt.Errorf("no active instance configured; use 'owui instances use <name>' or pass --instance")
	}

	inst, ok := cfg.Instances[targetInstance]
	if !ok {
		return nil, fmt.Errorf("instance %q not found in config", targetInstance)
	}

	return api.NewClient(inst.URL, inst.APIKey, cfg.Settings.TimeoutSeconds), nil
}

// ResolveClientForCompletion is a best-effort version for shell completion
// callbacks. It honors the --instance flag like ResolveClient so completions
// target the same instance the command will run against.
func ResolveClientForCompletion(cmd *cobra.Command) *api.Client {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	inst, ok := cfg.Instances[targetInstanceName(cmd, cfg)]
	if !ok {
		return nil
	}
	return api.NewClient(inst.URL, inst.APIKey, cfg.Settings.TimeoutSeconds)
}

// CmdContext returns the command's context, falling back to context.Background().
func CmdContext(cmd *cobra.Command) context.Context {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}
