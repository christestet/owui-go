package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/christestet/owui-go/internal/cli"
	"github.com/christestet/owui-go/internal/cli/root"
	"github.com/spf13/cobra/doc"
)

func main() {
	out := flag.String("out", "./docs/cli", "Output directory")
	format := flag.String("format", "markdown", "markdown|man|rest")
	front := flag.Bool("frontmatter", true, "Prepend YAML front matter to markdown")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	cli.RegisterAll(root.Cmd)

	cmd := root.Cmd
	cmd.DisableAutoGenTag = true

	switch *format {
	case "markdown":
		if *front {
			prep := func(filename string) string {
				base := filepath.Base(filename)
				name := strings.TrimSuffix(base, filepath.Ext(base))
				title := strings.ReplaceAll(name, "_", " ")

				return fmt.Sprintf("---\ntitle: %q\ndescription: \"Reference and examples for the %s command.\"\n---\n\n", title, title)
			}
			link := func(name string) string { return strings.ToLower(name) }

			if err := doc.GenMarkdownTreeCustom(cmd, *out, prep, link); err != nil {
				log.Fatal(err)
			}
		} else {
			if err := doc.GenMarkdownTree(cmd, *out); err != nil {
				log.Fatal(err)
			}
		}
	case "man":
		if err := doc.GenManTree(cmd, &doc.GenManHeader{
			Title:   "OWUI",
			Section: "1",
		}, *out); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unsupported format: %s", *format)
	}

	fmt.Printf("Documentation generated in %s (format: %s)\n", *out, *format)
}
