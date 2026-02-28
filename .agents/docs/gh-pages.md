Here is the streamlined, instruction-based manifest in 
English. You can copy and paste this directly to an LLM 
or developer for immediate implementation. 📋 LLM Prompt: 
Cobra CLI Docs -> MkDocs Pipeline (2026 Standard) Context 
for the Executing Agent: Implement an automated pipeline 
that generates CLI documentation from a Go Cobra project, 
builds it using MkDocs (Material Theme), and deploys it 
via GitHub Actions using the modern Pages Artifact upload 
method (no gh-pages branch). The Markdown output must 
include semantic YAML frontmatter optimized for search 
indexes, MkDocs plugins, and LLM ingestion. Stack 
Requirements:
 * Go: 1.26.0 * Python: 3.14 * Deployment: GitHub Actions 
 Native Artifact Upload
Execute the following 4 file creations/modifications 
exactly as written. Pay close attention to the 
[CUSTOMIZATION POINT] markers. 1. Create the Document 
Generator (internal/tools/docgen/main.go) This script 
walks the Cobra tree, outputs Markdown, and injects 
MkDocs-compatible YAML frontmatter (including automated 
tags based on the command path). package main import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"github.com/spf13/cobra/doc"
	// [CUSTOMIZATION POINT]: Replace 
"example.com/myapp" with the actual module path
	"example.com/myapp/cmd" ) func main() {
	out := flag.String("out", "./docs/cli", "Output 
directory")
	format := flag.String("format", "markdown", 
"markdown|man|rest")
	front := flag.Bool("frontmatter", true, "Prepend 
YAML front matter to markdown")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	root := cmd.Root()
	// CRITICAL: Disable auto-generated timestamps to 
prevent CI cache churn and LLM vector invalidation
	root.DisableAutoGenTag = true
	switch *format {
	case "markdown":
		if *front {
			prep := func(filename string) 
string {
				base := 
filepath.Base(filename)
				name := 
strings.TrimSuffix(base, filepath.Ext(base))
				title := 
strings.ReplaceAll(name, "_", " ")
				
				// Generate MkDocs tags 
from the filename (e.g., myapp_serve -> tags: \n - myapp 
\n - serve) arts := strings.Split(name, "_")
				tags := ""
				for _, part := range 
parts {
					tags += 
fmt.Sprintf("\n - %s", part)
				}
				return 
fmt.Sprintf("---\ntitle: %q\nslug: %q\ndescription: 
\"Reference and examples for command: 
%s\"\ntags:%s\n---\n\n", title, name, title, tags)
			}
			link := func(name string) string 
{ return strings.ToLower(name) }
			
if err := doc.GenMarkdownTreeCustom(root, *out, prep, 
link); err != nil {r)r := doc.GenMarkdownTree(root, 
*out); err != nil {r) ported format: %s", *format)
	} } 2. Configure MkDocs (mkdocs.yml) Enable 
native Material features and the tags plugin to consume 
the frontmatter generated in Step 1. # [CUSTOMIZATION 
POINT]: Update site name and description site_name: My 
CLI Documentation site_description: "Official CLI 
reference, optimized for humans and LLMs" theme:
  name: material features:
    - navigation.instant - navigation.tracking - 
    navigation.tabs - search.suggest - search.highlight
plugins:
  - search - tags - social nav: - Home: index.md # 
  [CUSTOMIZATION POINT]: Replace "myapp.md" with your 
  actual root command markdown file - CLI Reference: 
  cli/myapp.md
3. Create the CI/CD Pipeline (.github/workflows/docs.yml) 
Use the modern Artifact deployment strategy. Hardcode Go 
to 1.26.0 and Python to 3.14. name: Build and Deploy 
MkDocs on:
  push:
    branches: ["main"] # CRITICAL: Required for native 
GitHub Pages Artifact Deployment permissions:
  contents: read pages: write id-token: write jobs: 
  build-deploy:
    runs-on: ubuntu-latest environment:
      name: github-pages url: ${{ 
      steps.deployment.outputs.page_url }}
    >>>> >>>> steps:
 Go
        uses: actions/setup-go@v5 with:
          go-version: '1.26.0' cache: true
      - name: Set up Python
        uses: actions/setup-python@v5 with:
          python-version: '3.14' cache: 'pip'
      - name: Generate Cobra CLI Docs
        run: go run ./internal/tools/docgen -out 
./docs/cli -format markdown -frontmatter
      - name: Install MkDocs & Material Theme
        run: pip install mkdocs-material
      - name: Build MkDocs Site
        run: mkdocs build # Builds to ./site directory
      - name: Upload GitHub Pages Artifact
        uses: actions/upload-pages-artifact@v3 with:
          path: ./site
      - name: Deploy to GitHub Pages
        id: deployment uses: actions/deploy-pages@v4 4. 
Update Ignore Rules (.gitignore) Ensure the CI-generated 
files do not pollute the repository. # Exclude 
auto-generated CLI docs docs/cli/*.md 
