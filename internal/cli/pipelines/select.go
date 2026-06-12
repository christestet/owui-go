package pipelines

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/cli/prompts"
)

func selectIDWithURLIdx(prompt string, options []huh.Option[string], invalidValueMsg, invalidIdxMsg string) (string, *int, error) {
	selected := ""
	if err := prompts.RunSearchableSelect(prompt, options, &selected); err != nil {
		return "", nil, prompts.WrapInteractiveCancelled(err)
	}
	parts := strings.SplitN(selected, "|", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("%s: %q", invalidValueMsg, selected)
	}
	idx, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", invalidIdxMsg, err)
	}
	return parts[0], &idx, nil
}
