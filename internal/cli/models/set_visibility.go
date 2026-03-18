package models

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/christestet/owui-go/internal/api"
	"github.com/spf13/cobra"
)

var visibilityAction = &batchModelAction{
	validActions: []string{"public", "private"},
	actionLabel:  "visibility",
	filterFn: func(m api.ModelAccessResponse, action string) bool {
		if action == "public" {
			return isPrivate(m)
		}
		return isPublic(m)
	},
	optionLabelFn: func(m api.ModelAccessResponse) string {
		label := fmt.Sprintf("%s (%s)", m.Name, m.ID)
		if isPrivate(m) {
			label += fmt.Sprintf(" [%d grants]", len(m.AccessGrants))
		}
		return label
	},
	confirmMsgFn: func(action string, count int, names []string) string {
		return fmt.Sprintf("Confirm making %d model(s) %s: %s?", count, action, strings.Join(names, ", "))
	},
	applyFn: func(ctx context.Context, client *api.Client, m api.ModelAccessResponse, action string, w io.Writer) error {
		grants := []api.AccessGrantModel{}
		if action == "public" && len(m.AccessGrants) > 0 {
			fmt.Fprintf(w, "Note: This will remove all %d access grants from '%s'.\n", len(m.AccessGrants), m.Name)
		}

		form := api.ModelAccessGrantsForm{
			ID:           m.ID,
			Name:         m.Name,
			AccessGrants: grants,
		}
		if err := client.UpdateModelAccess(ctx, form); err != nil {
			return fmt.Errorf("failed to set visibility for model '%s': %w", m.Name, err)
		}

		if action == "public" {
			fmt.Fprintf(w, "Successfully set model '%s' to public\n", m.Name)
		} else {
			fmt.Fprintf(w, "Successfully set model '%s' to private (admin-only access)\n", m.Name)
			fmt.Fprintln(w, "Tip: Use 'owui models add-to-group' to grant access to specific groups.")
		}
		return nil
	},
	completionFilterFn: func(action string) func(api.ModelAccessResponse) bool {
		switch action {
		case "public":
			return func(m api.ModelAccessResponse) bool { return isPrivate(m) }
		case "private":
			return func(m api.ModelAccessResponse) bool { return isPublic(m) }
		default:
			return nil
		}
	},
}

var setVisibilityCmd = &cobra.Command{
	Use:               "set-visibility <public|private> [model_id ...]",
	Short:             "Set model visibility to public or private",
	Long:              `Change the visibility of one or more models between public and private.`,
	Args:              cobra.MinimumNArgs(0),
	ValidArgsFunction: visibilityAction.validArgsFunction,
	RunE:              visibilityAction.runE,
}
