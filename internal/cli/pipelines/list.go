package pipelines

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List pipeline registrations and pipes",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		inv, err := buildInventory(ctx, client, cmd.ErrOrStderr())
		if err != nil {
			return err
		}

		filterText, _ := cmd.Flags().GetString("filter")
		inv = filterInventory(inv, filterText)

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			b, err := json.MarshalIndent(inv, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return err
		}

		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Registered Pipelines"); err != nil {
			return err
		}
		wr := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		if _, err := fmt.Fprintln(wr, "REGISTRATION ID\tURL\tURL IDX\tPIPES"); err != nil {
			return err
		}
		for _, r := range inv.Registrations {
			if _, err := fmt.Fprintf(wr, "%s\t%s\t%d\t%d\n", r.RegistrationID, r.URL, r.URLIdx, r.PipeCount); err != nil {
				return err
			}
		}
		if err := wr.Flush(); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Pipes"); err != nil {
			return err
		}
		wp := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		if _, err := fmt.Fprintln(wp, "PIPE ID\tNAME\tREGISTRATION ID\tURL IDX"); err != nil {
			return err
		}
		for _, p := range inv.Pipes {
			if _, err := fmt.Fprintf(wp, "%s\t%s\t%s\t%d\n", p.PipeID, p.Name, p.RegistrationID, p.URLIdx); err != nil {
				return err
			}
		}
		if err := wp.Flush(); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d registration(s), %d pipe(s).\n", len(inv.Registrations), len(inv.Pipes))
		return err
	},
}
