package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/christestet/owui-go/internal/cli"
	"github.com/christestet/owui-go/internal/cli/root"
	"github.com/spf13/cobra"
)

const (
	beginMarker = "<!-- BEGIN:CLI_COMMANDS -->"
	endMarker   = "<!-- END:CLI_COMMANDS -->"
)

type commandDoc struct {
	Path  string
	Short string
}

func main() {
	readmePath := flag.String("readme", "./README.md", "Path to README file to update")
	flag.Parse()

	cli.RegisterAll(root.Cmd)

	commands := collectCommands(root.Cmd)
	content, err := os.ReadFile(*readmePath)
	if err != nil {
		log.Fatalf("read readme: %v", err)
	}

	updated, err := replaceCLISection(string(content), renderCLISection(commands))
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(*readmePath, []byte(updated), 0o644); err != nil {
		log.Fatalf("write readme: %v", err)
	}

	fmt.Printf("Updated %s with %d commands\n", *readmePath, len(commands))
}

func collectCommands(rootCmd *cobra.Command) []commandDoc {
	var docs []commandDoc
	walkCommands(rootCmd, &docs)
	slices.SortStableFunc(docs, func(a, b commandDoc) int {
		return strings.Compare(a.Path, b.Path)
	})
	return docs
}

func walkCommands(cmd *cobra.Command, docs *[]commandDoc) {
	if shouldInclude(cmd) {
		short := strings.TrimSpace(cmd.Short)
		if short == "" {
			short = "(no description)"
		}
		*docs = append(*docs, commandDoc{
			Path:  cmd.CommandPath(),
			Short: short,
		})
	}

	for _, sub := range cmd.Commands() {
		walkCommands(sub, docs)
	}
}

func shouldInclude(cmd *cobra.Command) bool {
	if cmd.Hidden || cmd.Deprecated != "" {
		return false
	}
	if cmd.Name() == "help" {
		return false
	}
	return cmd.IsAvailableCommand()
}

func renderCLISection(commands []commandDoc) string {
	var b strings.Builder
	b.WriteString("### CLI Command Reference (auto-generated)\n\n")
	b.WriteString("Run `make docs-readme` to refresh this section.\n\n")
	b.WriteString("| Command | Description |\n")
	b.WriteString("| --- | --- |\n")
	for _, c := range commands {
		b.WriteString(fmt.Sprintf("| `%s` | %s |\n", c.Path, sanitizeTable(c.Short)))
	}
	return b.String()
}

func replaceCLISection(readme, generated string) (string, error) {
	start := strings.Index(readme, beginMarker)
	if start == -1 {
		return "", fmt.Errorf("missing marker: %s", beginMarker)
	}
	end := strings.Index(readme, endMarker)
	if end == -1 {
		return "", fmt.Errorf("missing marker: %s", endMarker)
	}
	if end <= start {
		return "", fmt.Errorf("invalid marker ordering")
	}

	head := readme[:start+len(beginMarker)]
	tail := readme[end:]
	return head + "\n\n" + strings.TrimSpace(generated) + "\n\n" + tail, nil
}

func sanitizeTable(s string) string {
	out := strings.ReplaceAll(s, "|", "\\|")
	return strings.ReplaceAll(out, "\n", " ")
}
