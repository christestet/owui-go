package groups

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

const (
	interactiveSelectHeight = 15
	searchHintSuffix        = " (press / to search)"
	errInteractiveCancelled = "interactive input cancelled: %w"
)

func withSearchHint(title string) string {
	return title + searchHintSuffix
}

func wrapInteractiveCancelled(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(errInteractiveCancelled, err)
}

func runSearchableSelect(title string, options []huh.Option[string], value *string) error {
	return huh.NewSelect[string]().
		Title(withSearchHint(title)).
		Options(options...).
		Height(interactiveSelectHeight).
		Value(value).
		Run()
}

func runSearchableMultiSelect(title string, options []huh.Option[string], value *[]string) error {
	return huh.NewMultiSelect[string]().
		Title(withSearchHint(title)).
		Options(options...).
		Filterable(true).
		Height(interactiveSelectHeight).
		Value(value).
		Run()
}
