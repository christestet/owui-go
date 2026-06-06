package pipelines

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a pipeline file",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		filePath, _ := cmd.Flags().GetString("file")
		interactive := false
		if filePath == "" {
			interactive = true
			err := huh.NewInput().Title("Path to pipeline file").Value(&filePath).Run()
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}
		if filePath == "" {
			return fmt.Errorf("--file is required")
		}
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("invalid --file %q: %w", filePath, err)
		}

		inv, err := buildInventory(ctx, client, nil)
		if err != nil {
			return err
		}

		urlIdx, _ := cmd.Flags().GetInt("url-idx")
		if !cmd.Flags().Changed("url-idx") {
			customSet := false
			if interactive {
				custom, err := prompts.ConfirmYN(cmd.InOrStdin(), cmd.OutOrStdout(), "Use custom urlIdx?")
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
						return prompts.WrapInteractiveCancelled(err)
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

		resp, err := client.UploadPipelineFileRaw(ctx, filePath, urlIdx)
		if err != nil {
			return err
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			b, err := json.MarshalIndent(map[string]any{
				"file":          filePath,
				"url_idx":       urlIdx,
				"upload_result": resp,
			}, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully uploaded pipeline file '%s' to urlIdx=%d\n", filePath, urlIdx)
		return nil
	},
}

func init() {
	uploadCmd.Flags().String("file", "", "path to pipeline file")
	uploadCmd.Flags().Int("url-idx", 0, "urlIdx to assign (default: next free index)")
}
