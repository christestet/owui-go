package cli

import (
	"github.com/christestet/owui-go/internal/cli/completion"
	"github.com/christestet/owui-go/internal/cli/groups"
	"github.com/christestet/owui-go/internal/cli/instances"
	cliUpdate "github.com/christestet/owui-go/internal/cli/update"
	"github.com/christestet/owui-go/internal/cli/users"
	cliVersion "github.com/christestet/owui-go/internal/cli/version"
	"github.com/spf13/cobra"
)

// RegisterAll registers all subcommands on the given root command.
func RegisterAll(root *cobra.Command) {
	root.AddCommand(cliVersion.Cmd)
	root.AddCommand(cliUpdate.Cmd)
	root.AddCommand(completion.Cmd)
	instances.Register(root)
	users.Register(root)
	groups.Register(root)
}
