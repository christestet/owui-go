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
	Long: `To load completions:

Bash:

  $ source <(owui completion bash)

  # To load completions for each session, add to your .bashrc:
  # { owui completion bash; } > /etc/bash_completion.d/owui

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ owui completion zsh > "${fpath[1]}/_owui"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ owui completion fish | source

  # To load completions for each session, execute once:
  $ owui completion fish > ~/.config/fish/completions/owui.fish

PowerShell:

  PS> owui completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> owui completion powershell > owui.ps1
  # and source this file from your PowerShell profile.
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
		// Standard user-level fpath entries
		zshDir := filepath.Join(home, ".zsh", "completions")
		if err := os.MkdirAll(zshDir, 0755); err != nil {
			// Fallback to ~/.zfunc
			zshDir = filepath.Join(home, ".zfunc")
			if err := os.MkdirAll(zshDir, 0755); err != nil {
				return err
			}
		}
		dest := filepath.Join(zshDir, "_owui")
		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer f.Close()
		return root.GenZshCompletion(f)

	case "bash":
		// Standard user completions directory
		bashDir := filepath.Join(home, ".local", "share", "bash-completion", "completions")
		if err := os.MkdirAll(bashDir, 0755); err != nil {
			// Fallback to ~/.bash_completion.d
			bashDir = filepath.Join(home, ".bash_completion.d")
			if err := os.MkdirAll(bashDir, 0755); err != nil {
				bashDir = home
			}
		}
		dest := filepath.Join(bashDir, "owui")
		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer f.Close()
		return root.GenBashCompletion(f)

	case "fish":
		fishDir := filepath.Join(home, ".config", "fish", "completions")
		if err := os.MkdirAll(fishDir, 0755); err != nil {
			return err
		}
		dest := filepath.Join(fishDir, "owui.fish")
		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer f.Close()
		return root.GenFishCompletion(f, true)

	default:
		return fmt.Errorf("unsupported shell for automatic installation: %s", shell)
	}
}
