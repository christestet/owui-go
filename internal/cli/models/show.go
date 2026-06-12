package models

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/cli/prompts"
	"github.com/christestet/owui-go/internal/cli/shared"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).PaddingLeft(2)
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(2)
	labelStyle    = lipgloss.NewStyle().Width(18).Foreground(lipgloss.Color("245")).PaddingLeft(2)
	enabledStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	disabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	privateStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	publicStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	dividerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(2)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	yesStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		width = 80
	}
	if width > 60 {
		width = 60
	}
	return width
}

func renderSectionDivider(label string, width int) string {
	prefix := "── " + label + " "
	remaining := width - len(prefix)
	if remaining < 0 {
		remaining = 0
	}
	return dividerStyle.Render(prefix + strings.Repeat("─", remaining))
}

func renderKeyValue(label, value string) string {
	return labelStyle.Render(label) + value
}

func renderStatusValue(isActive bool) string {
	if isActive {
		return enabledStyle.Render("enabled")
	}
	return disabledStyle.Render("disabled")
}

func renderVisibilityValue(m api.ModelAccessResponse) string {
	if isPublic(m) {
		return publicStyle.Render("public")
	}
	suffix := fmt.Sprintf(" (%d group grants)", len(m.AccessGrants))
	return privateStyle.Render("private") + suffix
}

func renderBoolValue(v *bool) string {
	if v != nil && *v {
		return yesStyle.Render("yes")
	}
	return dimStyle.Render("no")
}

var showCmd = &cobra.Command{
	Use:               "show [model_id]",
	Short:             "Show model details",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: modelCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := shared.ResolveClient(cmd)
		if err != nil {
			return err
		}

		ctx := shared.CmdContext(cmd)

		var modelID string
		if len(args) > 0 {
			modelID = args[0]
		} else {
			// Interactive select
			allModels, err := client.ListModels(ctx)
			if err != nil {
				return err
			}
			if len(allModels) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No models found.")
				return err
			}
			options := make([]huh.Option[string], 0, len(allModels))
			for _, m := range allModels {
				options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", m.Name, m.ID), m.ID))
			}
			err = prompts.RunSearchableSelect("Select model", options, &modelID)
			if err != nil {
				return prompts.WrapInteractiveCancelled(err)
			}
		}

		model, err := client.GetModel(ctx, modelID)
		if err != nil {
			return err
		}

		// Fetch groups for resolution
		groups, err := client.ListGroups(ctx)
		if err != nil {
			// Non-fatal: we can still show the model without group names
			groups = nil
		}
		groupMap := make(map[string]string)
		for _, g := range groups {
			groupMap[g.ID] = g.Name
		}

		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == "json" {
			return renderShowJSON(cmd, model, groupMap)
		}

		return renderShowPretty(cmd, model, groupMap)
	},
}

func renderShowJSON(cmd *cobra.Command, model *api.ModelAccessResponse, groupMap map[string]string) error {
	// Build a JSON-friendly map with resolved_groups
	type jsonGrant struct {
		api.AccessGrantModel
		ResolvedGroupName string `json:"resolved_group_name,omitempty"`
	}

	type jsonOutput struct {
		ID           string         `json:"id"`
		Name         string         `json:"name"`
		BaseModelID  string         `json:"base_model_id,omitempty"`
		IsActive     bool           `json:"is_active"`
		Meta         api.ModelMeta  `json:"meta"`
		AccessGrants []jsonGrant    `json:"access_grants"`
		User         *api.ModelUser `json:"user,omitempty"`
		CreatedAt    int64          `json:"created_at,omitempty"`
		UpdatedAt    int64          `json:"updated_at,omitempty"`
	}

	grants := make([]jsonGrant, 0, len(model.AccessGrants))
	for _, g := range model.AccessGrants {
		jg := jsonGrant{AccessGrantModel: g}
		if name, ok := groupMap[g.PrincipalID]; ok {
			jg.ResolvedGroupName = name
		}
		grants = append(grants, jg)
	}

	out := jsonOutput{
		ID:           model.ID,
		Name:         model.Name,
		BaseModelID:  model.BaseModelID,
		IsActive:     model.IsActive,
		Meta:         model.Meta,
		AccessGrants: grants,
		User:         model.User,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return err
}

func renderShowPretty(cmd *cobra.Command, model *api.ModelAccessResponse, groupMap map[string]string) error {
	out := cmd.OutOrStdout()
	width := getTerminalWidth()
	printLine := func(args ...any) error {
		_, err := fmt.Fprintln(out, args...)
		return err
	}

	// Header
	if err := printLine(); err != nil {
		return err
	}
	if err := printLine(titleStyle.Render(model.Name)); err != nil {
		return err
	}
	if err := printLine(subtitleStyle.Render(model.ID)); err != nil {
		return err
	}

	// General section
	if err := printLine(); err != nil {
		return err
	}
	if err := printLine(renderSectionDivider("General", width)); err != nil {
		return err
	}
	if err := printLine(renderKeyValue("Base Model", model.BaseModelID)); err != nil {
		return err
	}
	if err := printLine(renderKeyValue("Description", model.Meta.Description)); err != nil {
		return err
	}
	if model.User != nil {
		owner := fmt.Sprintf("%s (%s)", model.User.Name, model.User.Email)
		if err := printLine(renderKeyValue("Owner", owner)); err != nil {
			return err
		}
	}
	if model.CreatedAt > 0 {
		if err := printLine(renderKeyValue("Created", time.Unix(model.CreatedAt, 0).Format("2006-01-02 15:04:05"))); err != nil {
			return err
		}
	}
	if model.UpdatedAt > 0 {
		if err := printLine(renderKeyValue("Updated", time.Unix(model.UpdatedAt, 0).Format("2006-01-02 15:04:05"))); err != nil {
			return err
		}
	}

	// Status section
	if err := printLine(); err != nil {
		return err
	}
	if err := printLine(renderSectionDivider("Status", width)); err != nil {
		return err
	}
	if err := printLine(renderKeyValue("Status", renderStatusValue(model.IsActive))); err != nil {
		return err
	}
	if err := printLine(renderKeyValue("Visibility", renderVisibilityValue(*model))); err != nil {
		return err
	}

	// Capabilities section
	if err := printLine(); err != nil {
		return err
	}
	if err := printLine(renderSectionDivider("Capabilities", width)); err != nil {
		return err
	}
	if err := printLine(renderKeyValue("Vision", renderBoolValue(model.Meta.Capabilities.Vision))); err != nil {
		return err
	}
	if err := printLine(renderKeyValue("Citations", renderBoolValue(model.Meta.Capabilities.Citations))); err != nil {
		return err
	}
	if err := printLine(renderKeyValue("Code Interpreter", renderBoolValue(model.Meta.Capabilities.CodeInterpreter))); err != nil {
		return err
	}

	// Access Grants section
	if err := printLine(); err != nil {
		return err
	}
	if err := printLine(renderSectionDivider("Access Grants", width)); err != nil {
		return err
	}
	if len(model.AccessGrants) == 0 {
		if err := printLine(dimStyle.PaddingLeft(2).Render("No access grants -- model is public.")); err != nil {
			return err
		}
	} else {
		w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
		if _, err := fmt.Fprintln(w, "  GROUP\tPERMISSION\tGRANTED"); err != nil {
			return err
		}
		for _, grant := range model.AccessGrants {
			groupName := grant.PrincipalID
			if name, ok := groupMap[grant.PrincipalID]; ok {
				groupName = name
			}
			granted := "-"
			if grant.CreatedAt > 0 {
				granted = time.Unix(grant.CreatedAt, 0).Format("2006-01-02")
			}
			if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\n", groupName, grant.Permission, granted); err != nil {
				return err
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}

	return printLine()
}
