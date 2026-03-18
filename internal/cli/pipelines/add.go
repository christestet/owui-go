package pipelines

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a pipeline registration",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := resolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		urlValue, _ := cmd.Flags().GetString("url")
		interactive := false
		if urlValue == "" {
			interactive = true
			err := huh.NewInput().Title("Pipeline URL").Value(&urlValue).Run()
			if err != nil {
				return wrapInteractiveCancelled(err)
			}
		}
		if urlValue == "" {
			return fmt.Errorf("--url is required")
		}

		inv, err := buildInventory(ctx, client, nil)
		if err != nil {
			return err
		}

		urlIdx, _ := cmd.Flags().GetInt("url-idx")
		if !cmd.Flags().Changed("url-idx") {
			customSet := false
			if interactive {
				custom, err := prompts.ConfirmYN("Use custom urlIdx?")
				if err != nil {
					return err
				}
				if custom {
					idxText := ""
					err := huh.NewInput().
						Title("urlIdx").
						Value(&idxText).
						Run()
					if err != nil {
						return wrapInteractiveCancelled(err)
					}
					parsed, err := strconv.Atoi(idxText)
					if err != nil {
						return fmt.Errorf("invalid urlIdx %q: %w", idxText, err)
					}
					urlIdx = parsed
					customSet = true
				}
			}
			if !cmd.Flags().Changed("url-idx") && !customSet {
				urlIdx = nextFreeURLIdx(inv)
			}
		}

		resp, err := client.AddPipelineRegistrationRaw(ctx, api.AddPipelineForm{URL: urlValue, URLIdx: urlIdx})
		if err != nil {
			return err
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			b, err := json.MarshalIndent(map[string]any{
				"url":          urlValue,
				"url_idx":      urlIdx,
				"registration": resp,
			}, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully added pipeline registration for '%s' (urlIdx=%d)\n", urlValue, urlIdx)
		return nil
	},
}

func init() {
	addCmd.Flags().String("url", "", "pipeline URL")
	addCmd.Flags().Int("url-idx", 0, "urlIdx to assign (default: next free index)")
}
