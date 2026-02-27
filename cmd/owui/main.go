package main

import (
	"github.com/christestet/owui-go/internal/cli/completion"
	"github.com/christestet/owui-go/internal/cli/instances"
	"github.com/christestet/owui-go/internal/cli/root"
	cliUpdate "github.com/christestet/owui-go/internal/cli/update"
	"github.com/christestet/owui-go/internal/cli/users"
	cliVersion "github.com/christestet/owui-go/internal/cli/version"
)

func main() {
	// Register subcommands
	root.Cmd.AddCommand(cliVersion.Cmd)
	root.Cmd.AddCommand(cliUpdate.Cmd)
	root.Cmd.AddCommand(completion.Cmd)
	instances.Register(root.Cmd)
	users.Register(root.Cmd)

	// Execute the root command
	root.Execute()
}
