package pipelines

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:               "remove [registration_id]",
	Short:             "Remove a pipeline registration",
	Aliases:           []string{"rm"},
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: registrationCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := resolveClient(cmd)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		inv, err := buildInventory(ctx, client, nil)
		if err != nil {
			return err
		}
		if len(inv.Registrations) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No pipeline registrations found.")
			return nil
		}

		var registrationID string
		var explicitIdx *int
		if cmd.Flags().Changed("url-idx") {
			idx, _ := cmd.Flags().GetInt("url-idx")
			explicitIdx = &idx
		}

		if len(args) > 0 {
			registrationID = args[0]
		} else {
			selected := ""
			options := make([]huh.Option[string], 0, len(inv.Registrations))
			for _, r := range inv.Registrations {
				label := fmt.Sprintf("%s (idx=%d, %s)", r.RegistrationID, r.URLIdx, r.URL)
				options = append(options, huh.NewOption(label, fmt.Sprintf("%s|%d", r.RegistrationID, r.URLIdx)))
			}
			err := runSearchableSelect("Select registration to delete", options, &selected)
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
			parts := strings.SplitN(selected, "|", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid selected registration value: %q", selected)
			}
			registrationID = parts[0]
			idx, err := strconv.Atoi(parts[1])
			if err != nil {
				return fmt.Errorf("invalid selected registration urlIdx: %w", err)
			}
			explicitIdx = &idx
		}

		reg, err := resolveRegistration(inv, registrationID, explicitIdx)
		if err != nil {
			return err
		}

		confirmed, err := prompts.ConfirmYN(fmt.Sprintf("Confirm deleting registration '%s' (urlIdx=%d)?", reg.RegistrationID, reg.URLIdx))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}

		if err := client.DeletePipelineRegistration(ctx, api.DeletePipelineForm{ID: reg.RegistrationID, URLIdx: reg.URLIdx}); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted pipeline registration '%s'\n", reg.RegistrationID)
		return nil
	},
}

func init() {
	removeCmd.Flags().Int("url-idx", 0, "override urlIdx when resolving duplicate registration IDs")
}
