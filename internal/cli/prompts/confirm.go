package prompts

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ConfirmYN writes the question followed by "[y/N]: " to out and returns true if
// the user types "y" or "Y" on in. Any other input (including empty/Enter) is
// treated as "no". Taking in/out as parameters keeps the prompt redirectable and
// testable; callers with a *cobra.Command pass cmd.InOrStdin()/cmd.OutOrStdout().
func ConfirmYN(in io.Reader, out io.Writer, question string) (bool, error) {
	if _, err := fmt.Fprintf(out, "%s [y/N]: ", question); err != nil {
		return false, err
	}
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, nil
	}
	return strings.EqualFold(strings.TrimSpace(scanner.Text()), "y"), nil
}
