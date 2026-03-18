package shared

import (
	"context"
	"fmt"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/cobra"
)

// ResolveClient loads config, resolves the target instance, and returns an API client.
func ResolveClient(cmd *cobra.Command) (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	targetInstance, _ := cmd.Flags().GetString("instance")
	if targetInstance == "" {
		targetInstance = cfg.ActiveInstance
	}
	if targetInstance == "" {
		return nil, fmt.Errorf("no active instance configured; use 'owui instances use <name>' or pass --instance")
	}

	inst, ok := cfg.Instances[targetInstance]
	if !ok {
		return nil, fmt.Errorf("instance %q not found in config", targetInstance)
	}

	return api.NewClient(inst.URL, inst.APIKey, cfg.Settings.TimeoutSeconds), nil
}

// ResolveClientForCompletion is a best-effort version for shell completion callbacks.
func ResolveClientForCompletion() *api.Client {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	inst, ok := cfg.Instances[cfg.ActiveInstance]
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
