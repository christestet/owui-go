package prompts

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmYN prints the question followed by "[y/N]: " and returns true if the user types "y" or "Y".
// Any other input (including empty/Enter) is treated as "no".
func ConfirmYN(question string) (bool, error) {
	fmt.Printf("%s [y/N]: ", question)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, nil
	}
	return strings.EqualFold(strings.TrimSpace(scanner.Text()), "y"), nil
}
