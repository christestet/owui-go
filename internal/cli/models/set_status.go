package models

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/christestet/owui-go/internal/api"
	"github.com/spf13/cobra"
)

var statusAction = &batchModelAction{
	validActions: []string{"enable", "disable"},
	actionLabel:  "action",
	filterFn: func(m api.ModelAccessResponse, action string) bool {
		if action == "enable" {
			return !m.IsActive
		}
		return m.IsActive
	},
	confirmMsgFn: func(action string, count int, names []string) string {
		verb := "enabling"
		if action == "disable" {
			verb = "disabling"
		}
		return fmt.Sprintf("Confirm %s %d model(s): %s?", verb, count, strings.Join(names, ", "))
	},
	applyFn: func(ctx context.Context, client *api.Client, m api.ModelAccessResponse, action string, w io.Writer) error {
		if _, err := client.ToggleModel(ctx, m.ID); err != nil {
			return fmt.Errorf("failed to %s model '%s': %w", action, m.Name, err)
		}
		fmt.Fprintf(w, "Successfully %sd model '%s'\n", action, m.Name)
		return nil
	},
	completionFilterFn: func(action string) func(api.ModelAccessResponse) bool {
		switch action {
		case "enable":
			return func(m api.ModelAccessResponse) bool { return !m.IsActive }
		case "disable":
			return func(m api.ModelAccessResponse) bool { return m.IsActive }
		default:
			return nil
		}
	},
}

var setStatusCmd = &cobra.Command{
	Use:               "set-status <enable|disable> [model_id ...]",
	Short:             "Enable or disable models",
	Long:              `Enable or disable one or more models by toggling their is_active state.`,
	Args:              cobra.MinimumNArgs(0),
	ValidArgsFunction: statusAction.validArgsFunction,
	RunE:              statusAction.runE,
}
