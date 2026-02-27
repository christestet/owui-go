package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/cobra"
)

var (
	quiet bool

	// UserHomeDirFunc allows mocking os.UserHomeDir in tests
	UserHomeDirFunc = os.UserHomeDir
)

// Cmd represents the completion command
var Cmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `Generate or install shell completion scripts.

Recommended: use 'owui completion install' to automatically install
completions for your current shell and configure your shell rc file.

  $ owui completion install

This detects your shell (bash/zsh/fish), writes the completion script,
and adds the necessary lines to your ~/.zshrc or ~/.bashrc.
Restart your shell afterwards.

Manual setup (generates completion script to stdout):

Bash:
  $ source <(owui completion bash)

Zsh:
  $ owui completion zsh > "${fpath[1]}/_owui"

Fish:
  $ owui completion fish | source

PowerShell:
  PS> owui completion powershell | Out-String | Invoke-Expression
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install completion script for the current shell",
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := detectShell()
		if shell == "" {
			return fmt.Errorf("could not detect shell")
		}

		if !quiet {
			fmt.Printf("Detected shell: %s\n", shell)
		}

		err := installCompletion(cmd.Root(), shell)
		if err != nil {
			if !quiet {
				fmt.Printf("Failed to install completion: %v\n", err)
				fmt.Println("Please follow the manual instructions in 'owui completion --help'")
			}
			return nil
		}

		cfg, err := config.Load()
		if err == nil {
			cfg.Cli.CompletionsInstalled = true
			_ = cfg.Save()
		}

		if !quiet {
			fmt.Printf("Successfully installed completion for %s\n", shell)
			fmt.Println("Please restart your shell or source your configuration to take effect.")
		}

		return nil
	},
}

func init() {
	installCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress output")
	Cmd.AddCommand(installCmd)
}

func detectShell() string {
	shellPath := os.Getenv("SHELL")
	if shellPath != "" {
		if strings.Contains(shellPath, "zsh") {
			return "zsh"
		}
		if strings.Contains(shellPath, "bash") {
			return "bash"
		}
		if strings.Contains(shellPath, "fish") {
			return "fish"
		}
	}
	return ""
}

func installCompletion(root *cobra.Command, shell string) error {
	home, err := UserHomeDirFunc()
	if err != nil {
		return err
	}

	switch shell {
	case "zsh":
		return installZshCompletion(root, home)
	case "bash":
		return installBashCompletion(root, home)
	case "fish":
		return installFishCompletion(root, home)
	default:
		return fmt.Errorf("unsupported shell for automatic installation: %s", shell)
	}
}

func installZshCompletion(root *cobra.Command, home string) error {
	zshDir := filepath.Join(home, ".zsh", "completions")
	if err := os.MkdirAll(zshDir, 0755); err != nil {
		zshDir = filepath.Join(home, ".zfunc")
		if err := os.MkdirAll(zshDir, 0755); err != nil {
			return err
		}
	}
	if err := writeCompletionFile(filepath.Join(zshDir, "_owui"), func(f *os.File) error {
		return root.GenZshCompletion(f)
	}); err != nil {
		return err
	}
	return ensureZshConfig(home)
}

func installBashCompletion(root *cobra.Command, home string) error {
	bashDir := filepath.Join(home, ".local", "share", "bash-completion", "completions")
	if err := os.MkdirAll(bashDir, 0755); err != nil {
		bashDir = filepath.Join(home, ".bash_completion.d")
		if err := os.MkdirAll(bashDir, 0755); err != nil {
			bashDir = home
		}
	}
	if err := writeCompletionFile(filepath.Join(bashDir, "owui"), func(f *os.File) error {
		return root.GenBashCompletion(f)
	}); err != nil {
		return err
	}
	return ensureBashConfig(home)
}

func installFishCompletion(root *cobra.Command, home string) error {
	fishDir := filepath.Join(home, ".config", "fish", "completions")
	if err := os.MkdirAll(fishDir, 0755); err != nil {
		return err
	}
	return writeCompletionFile(filepath.Join(fishDir, "owui.fish"), func(f *os.File) error {
		return root.GenFishCompletion(f, true)
	})
}

func writeCompletionFile(dest string, gen func(*os.File) error) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return gen(f)
}

const owuiShellMarker = "# owui shell completions"

func ensureZshConfig(home string) error {
	rcPath := filepath.Join(home, ".zshrc")

	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	rc := string(content)

	// Already configured
	if strings.Contains(rc, owuiShellMarker) {
		return nil
	}

	hasFpath := strings.Contains(rc, "~/.zsh/completions")
	hasCompinit := strings.Contains(rc, "compinit")

	if hasFpath && hasCompinit {
		return nil
	}

	var lines []string
	lines = append(lines, "", owuiShellMarker)
	if !hasFpath {
		lines = append(lines, `fpath=(~/.zsh/completions $fpath)`)
	}
	if !hasCompinit {
		lines = append(lines, "autoload -U compinit; compinit")
	}
	lines = append(lines, "")

	f, err := os.OpenFile(rcPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(strings.Join(lines, "\n"))
	if err != nil {
		return err
	}

	if !quiet {
		fmt.Printf("Added completion config to %s\n", rcPath)
	}
	return nil
}

func ensureBashConfig(home string) error {
	rcPath := filepath.Join(home, ".bashrc")

	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	rc := string(content)

	// Already configured
	if strings.Contains(rc, owuiShellMarker) {
		return nil
	}

	sourceLine := `[ -f ~/.local/share/bash-completion/completions/owui ] && source ~/.local/share/bash-completion/completions/owui`
	if strings.Contains(rc, sourceLine) {
		return nil
	}

	f, err := os.OpenFile(rcPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	block := fmt.Sprintf("\n%s\n%s\n", owuiShellMarker, sourceLine)
	_, err = f.WriteString(block)
	if err != nil {
		return err
	}

	if !quiet {
		fmt.Printf("Added completion config to %s\n", rcPath)
	}
	return nil
}
