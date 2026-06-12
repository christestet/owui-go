package pipelines

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:               "remove [registration_id]",
	Short:             "Remove a pipeline registration",
	Aliases:           []string{"rm"},
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: registrationCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}
		ctx := shared.CmdContext(cmd)

		inv, err := buildInventory(ctx, client, nil)
		if err != nil {
			return err
		}
		if len(inv.Registrations) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No pipeline registrations found.")
			return err
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
			options := make([]huh.Option[string], 0, len(inv.Registrations))
			for _, r := range inv.Registrations {
				label := fmt.Sprintf("%s (idx=%d, %s)", r.RegistrationID, r.URLIdx, r.URL)
				options = append(options, huh.NewOption(label, fmt.Sprintf("%s|%d", r.RegistrationID, r.URLIdx)))
			}
			var idx *int
			registrationID, idx, err = selectIDWithURLIdx("Select registration to delete", options, "invalid selected registration value", "invalid selected registration urlIdx")
			if err != nil {
				return err
			}
			explicitIdx = idx
		}

		reg, err := resolveRegistration(inv, registrationID, explicitIdx)
		if err != nil {
			return err
		}

		confirmed, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Confirm deleting registration '%s' (urlIdx=%d)?", reg.RegistrationID, reg.URLIdx))
		if err != nil {
			return err
		}
		if !confirmed {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return err
		}

		if err := client.DeletePipelineRegistration(ctx, api.DeletePipelineForm{ID: reg.RegistrationID, URLIdx: reg.URLIdx}); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted pipeline registration '%s'\n", reg.RegistrationID)
		return err
	},
}

func init() {
	removeCmd.Flags().Int("url-idx", 0, "override urlIdx when resolving duplicate registration IDs")
}
