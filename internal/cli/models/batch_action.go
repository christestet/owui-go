package models

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

// batchModelAction defines a reusable pattern for commands that apply an action
// to one or more models (e.g., enable/disable, public/private).
type batchModelAction struct {
	// validActions lists the allowed action strings (e.g., ["enable", "disable"]).
	validActions []string
	// actionLabel is used in error messages (e.g., "action", "visibility").
	actionLabel string
	// filterFn returns true for models eligible for the given action.
	filterFn func(m api.ModelAccessResponse, action string) bool
	// optionLabelFn formats the label for interactive selection.
	// If nil, defaults to "Name (ID)".
	optionLabelFn func(m api.ModelAccessResponse) string
	// confirmMsgFn builds the confirmation prompt.
	confirmMsgFn func(action string, count int, names []string) string
	// applyFn applies the action to a single model.
	applyFn func(ctx context.Context, client *api.Client, m api.ModelAccessResponse, action string, w io.Writer) error
	// completionFilterFn returns the model filter for shell completion based on the action.
	completionFilterFn func(action string) func(api.ModelAccessResponse) bool
}

func (b *batchModelAction) validArgsFunction(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		var comps []string
		for _, a := range b.validActions {
			if strings.HasPrefix(a, toComplete) {
				comps = append(comps, a)
			}
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	}
	action := args[0]
	filterFn := b.completionFilterFn(action)
	if filterFn == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return smartModelCompletionFunc(filterFn)(cmd, args[1:], toComplete)
}

func (b *batchModelAction) runE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s required: '%s'", b.actionLabel, strings.Join(b.validActions, "' or '"))
	}

	action := args[0]
	valid := false
	for _, a := range b.validActions {
		if action == a {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid %s %q: must be '%s'", b.actionLabel, action, strings.Join(b.validActions, "' or '"))
	}

	client, err := shared.ResolveClient(cmd)
	if err != nil {
		return err
	}

	ctx := shared.CmdContext(cmd)

	allModels, err := client.ListModels(ctx)
	if err != nil {
		return err
	}

	eligibleModels := shared.Filter(allModels, func(m api.ModelAccessResponse) bool {
		return b.filterFn(m, action)
	})

	modelIDs := args[1:]
	if len(modelIDs) == 0 {
		if len(eligibleModels) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No models to %s.\n", action)
			return nil
		}

		labelFn := b.optionLabelFn
		if labelFn == nil {
			labelFn = func(m api.ModelAccessResponse) string {
				return fmt.Sprintf("%s (%s)", m.Name, m.ID)
			}
		}

		options := make([]huh.Option[string], 0, len(eligibleModels))
		for _, m := range eligibleModels {
			options = append(options, huh.NewOption(labelFn(m), m.ID))
		}
		var selectedIDs []string
		err := prompts.RunSearchableMultiSelect(fmt.Sprintf("Select models to %s", action), options, &selectedIDs)
		if err != nil {
			return prompts.WrapInteractiveCancelled(err)
		}
		modelIDs = selectedIDs
	}

	if len(modelIDs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No models selected.")
		return nil
	}

	var resolvedModels []api.ModelAccessResponse
	for _, id := range modelIDs {
		m, err := findModelByNameOrID(allModels, id)
		if err != nil {
			return err
		}
		resolvedModels = append(resolvedModels, *m)
	}

	names := make([]string, 0, len(resolvedModels))
	for _, m := range resolvedModels {
		names = append(names, m.Name)
	}

	confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), b.confirmMsgFn(action, len(resolvedModels), names))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}

	for _, m := range resolvedModels {
		if err := b.applyFn(ctx, client, m, action, cmd.OutOrStdout()); err != nil {
			return err
		}
	}

	return nil
}
