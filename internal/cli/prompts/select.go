package prompts

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

const (
	InteractiveSelectHeight = 15
	SearchHintSuffix        = " (press / to search)"
	errInteractiveCancelled = "interactive input cancelled: %w"
)

func WithSearchHint(title string) string {
	return title + SearchHintSuffix
}

func WrapInteractiveCancelled(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(errInteractiveCancelled, err)
}

func RunSearchableSelect(title string, options []huh.Option[string], value *string) error {
	return huh.NewSelect[string]().
		Title(WithSearchHint(title)).
		Options(options...).
		Height(InteractiveSelectHeight).
		Value(value).
		Run()
}

func RunSearchableSelectWithDescription(title, description string, options []huh.Option[string], value *string) *huh.Select[string] {
	return huh.NewSelect[string]().
		Title(WithSearchHint(title)).
		Description(description).
		Options(options...).
		Height(InteractiveSelectHeight).
		Value(value)
}

func RunSearchableMultiSelect(title string, options []huh.Option[string], value *[]string) error {
	return huh.NewMultiSelect[string]().
		Title(WithSearchHint(title)).
		Options(options...).
		Filterable(true).
		Height(InteractiveSelectHeight).
		Value(value).
		Run()
}
