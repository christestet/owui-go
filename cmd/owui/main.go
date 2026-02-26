package main

import (
	"github.com/christestet/owui-go/internal/cli/instances"
	"github.com/christestet/owui-go/internal/cli/root"
	cliVersion "github.com/christestet/owui-go/internal/cli/version"
)

func main() {
	// Register subcommands
	root.Cmd.AddCommand(cliVersion.Cmd)
	instances.Register(root.Cmd)

	// Execute the root command
	root.Execute()
}
