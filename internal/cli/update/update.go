package update

import (
	"context"
	"fmt"
	"time"

	"github.com/christestet/owui-go/internal/cli/version"
	"github.com/christestet/owui-go/internal/updater"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "update",
	Short: "Update owui to the latest version",
	Long:  `Check for the latest release on GitHub and replace the running binary if a newer version is available.`,
	RunE:  runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	if version.Version == "dev" {
		fmt.Fprintln(cmd.OutOrStdout(), "Skipping update: running a development build.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Current version: %s\n", version.Version)
	fmt.Fprintln(cmd.OutOrStdout(), "Checking for updates...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	release, updateAvailable, err := updater.CheckLatest(ctx, version.Version)
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}
	if !updateAvailable {
		fmt.Fprintln(cmd.OutOrStdout(), "Already up to date.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "New version available: %s\nDownloading and applying update...\n", release.Version())
	if err := updater.Apply(ctx, release); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated to %s. Restart owui to use the new version.\n", release.Version())
	return nil
}
