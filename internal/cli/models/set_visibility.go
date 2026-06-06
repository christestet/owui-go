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
	validActions: []string{"public"},
	actionLabel:  "visibility",
	filterFn: func(m api.ModelAccessResponse, action string) bool {
		// Only currently-private models are eligible to be made public.
		return isPrivate(m)
	},
	optionLabelFn: func(m api.ModelAccessResponse) string {
		return fmt.Sprintf("%s (%s) [%d grants]", m.Name, m.ID, len(m.AccessGrants))
	},
	confirmMsgFn: func(action string, count int, names []string) string {
		return fmt.Sprintf("Confirm making %d model(s) public: %s?", count, strings.Join(names, ", "))
	},
	applyFn: func(ctx context.Context, client *api.Client, m api.ModelAccessResponse, action string, w io.Writer) error {
		if len(m.AccessGrants) > 0 {
			fmt.Fprintf(w, "Note: This will remove all %d access grants from '%s'.\n", len(m.AccessGrants), m.Name)
		}

		form := api.ModelAccessGrantsForm{
			ID:           m.ID,
			Name:         m.Name,
			AccessGrants: []api.AccessGrantModel{},
		}
		if err := client.UpdateModelAccess(ctx, form); err != nil {
			return fmt.Errorf("failed to set visibility for model '%s': %w", m.Name, err)
		}

		fmt.Fprintf(w, "Successfully set model '%s' to public\n", m.Name)
		return nil
	},
	completionFilterFn: func(action string) func(api.ModelAccessResponse) bool {
		if action == "public" {
			return func(m api.ModelAccessResponse) bool { return isPrivate(m) }
		}
		return nil
	},
}

var setVisibilityCmd = &cobra.Command{
	Use:   "set-visibility <public> [model_id ...]",
	Short: "Make models public (visible to everyone)",
	Long: `Make one or more models public by removing all of their access grants.

A model is "private" when it has one or more access grants restricting it to
specific groups or users. To make a model private, grant a group access with
'owui models add-to-group'; this command only removes grants to make a model
public again.`,
	Args:              cobra.MinimumNArgs(0),
	ValidArgsFunction: visibilityAction.validArgsFunction,
	RunE:              visibilityAction.runE,
}
