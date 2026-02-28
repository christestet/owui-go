package main

import (
	"github.com/christestet/owui-go/internal/cli"
	"github.com/christestet/owui-go/internal/cli/root"
)

func main() {
	cli.RegisterAll(root.Cmd)
	root.Execute()
}
