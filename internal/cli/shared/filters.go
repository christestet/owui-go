package shared

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/christestet/owui-go/internal/api"
)

// Filter returns items matching the predicate.
func Filter[T any](items []T, predicate func(T) bool) []T {
	var result []T
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// IsOAuthGroup returns true if the group was auto-created via OAuth.
func IsOAuthGroup(g api.Group) bool {
	return strings.HasPrefix(g.Description, "Group ") &&
		strings.HasSuffix(g.Description, " created automatically via OAuth.")
}

// FilterLocalGroups returns only non-OAuth groups.
func FilterLocalGroups(groups []api.Group) []api.Group {
	return Filter(groups, func(g api.Group) bool { return !IsOAuthGroup(g) })
}

// ResolveGroup looks up a group by exact ID or unique name.
func ResolveGroup(groups []api.Group, identifier string) (*api.Group, error) {
	for i := range groups {
		if groups[i].ID == identifier {
			return &groups[i], nil
		}
	}

	var matches []int
	for i := range groups {
		if groups[i].Name == identifier {
			matches = append(matches, i)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("group %q not found (use an exact group name or ID)", identifier)
	case 1:
		return &groups[matches[0]], nil
	default:
		return nil, fmt.Errorf("group %q is ambiguous (%d matches); use the group ID", identifier, len(matches))
	}
}

// FindGroupByName is kept for source compatibility and resolves IDs and unique names.
func FindGroupByName(groups []api.Group, identifier string) (*api.Group, error) {
	return ResolveGroup(groups, identifier)
}

// ResolveUser looks up a user by exact ID, unique email, or a unique username/name.
func ResolveUser(users []api.User, identifier string) (*api.User, error) {
	for i := range users {
		if users[i].ID == identifier {
			return &users[i], nil
		}
	}

	emailMatches := matchingUsers(users, func(u api.User) bool { return u.Email == identifier })
	if len(emailMatches) == 1 {
		return &users[emailMatches[0]], nil
	}
	if len(emailMatches) > 1 {
		return nil, fmt.Errorf("user email %q is ambiguous (%d matches); use the user ID", identifier, len(emailMatches))
	}

	identityMatches := matchingUsers(users, func(u api.User) bool {
		return u.Name == identifier || (u.Username != nil && *u.Username == identifier)
	})
	switch len(identityMatches) {
	case 0:
		return nil, fmt.Errorf("user %q not found (use an exact ID, email, username, or name)", identifier)
	case 1:
		return &users[identityMatches[0]], nil
	default:
		return nil, fmt.Errorf("user %q is ambiguous (%d matches); use the user's email or ID", identifier, len(identityMatches))
	}
}

func matchingUsers(users []api.User, match func(api.User) bool) []int {
	var matches []int
	seen := make(map[string]struct{})
	for i := range users {
		if !match(users[i]) {
			continue
		}
		if _, ok := seen[users[i].ID]; ok {
			continue
		}
		seen[users[i].ID] = struct{}{}
		matches = append(matches, i)
	}
	return matches
}

// FindUserByName is kept for source compatibility and resolves supported user identifiers.
func FindUserByName(users []api.User, identifier string) (*api.User, error) {
	return ResolveUser(users, identifier)
}

// ResolveUsers resolves identifiers and removes duplicate users while preserving order.
func ResolveUsers(users []api.User, identifiers []string) ([]api.User, error) {
	resolved := make([]api.User, 0, len(identifiers))
	seen := make(map[string]struct{}, len(identifiers))
	for _, identifier := range identifiers {
		user, err := ResolveUser(users, identifier)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[user.ID]; ok {
			continue
		}
		seen[user.ID] = struct{}{}
		resolved = append(resolved, *user)
	}
	return resolved, nil
}

// UserLabel returns a human-readable, unique user label when email is available.
func UserLabel(user api.User) string {
	if user.Email != "" {
		return fmt.Sprintf("%s (%s)", user.Name, user.Email)
	}
	if user.Username != nil && *user.Username != "" {
		return fmt.Sprintf("%s (@%s)", user.Name, *user.Username)
	}
	return fmt.Sprintf("%s (%s)", user.Name, user.ID)
}

// UserOptions builds human-readable interactive options carrying stable user IDs.
func UserOptions(users []api.User) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(users))
	for _, user := range users {
		options = append(options, huh.NewOption(UserLabel(user), user.ID))
	}
	return options
}

// GroupOptions builds human-readable interactive options carrying stable group IDs.
// IDs are displayed only when all available human metadata is identical.
func GroupOptions(groups []api.Group) []huh.Option[string] {
	baseLabels := make([]string, len(groups))
	counts := make(map[string]int, len(groups))
	for i, group := range groups {
		baseLabels[i] = groupBaseLabel(group)
		counts[baseLabels[i]]++
	}

	options := make([]huh.Option[string], 0, len(groups))
	for i, group := range groups {
		label := baseLabels[i]
		if counts[label] > 1 {
			label = fmt.Sprintf("%s [ID: %s]", label, group.ID)
		}
		options = append(options, huh.NewOption(label, group.ID))
	}
	return options
}

// UserCompletions returns unambiguous user identifiers with human-readable descriptions.
func UserCompletions(users []api.User, selected []string, toComplete string) []string {
	emailCounts := make(map[string]int, len(users))
	for _, user := range users {
		if user.Email != "" {
			emailCounts[user.Email]++
		}
	}
	selectedIDs := make(map[string]struct{}, len(selected))
	for _, identifier := range selected {
		if user, err := ResolveUser(users, identifier); err == nil {
			selectedIDs[user.ID] = struct{}{}
		}
	}
	var completions []string
	for _, user := range users {
		if _, ok := selectedIDs[user.ID]; ok {
			continue
		}
		matches := strings.HasPrefix(user.ID, toComplete) || strings.HasPrefix(user.Email, toComplete) || strings.HasPrefix(user.Name, toComplete)
		if user.Username != nil {
			matches = matches || strings.HasPrefix(*user.Username, toComplete)
		}
		if !matches {
			continue
		}
		identifier := user.Email
		if identifier == "" || emailCounts[identifier] > 1 {
			identifier = user.ID
		}
		completions = append(completions, fmt.Sprintf("%s\t%s", identifier, UserLabel(user)))
	}
	return completions
}

// GroupCompletions returns names when unique and IDs when duplicate names require disambiguation.
func GroupCompletions(groups []api.Group, selected []string, toComplete string) []string {
	nameCounts := make(map[string]int, len(groups))
	for _, group := range groups {
		nameCounts[group.Name]++
	}
	selectedIDs := make(map[string]struct{}, len(selected))
	for _, identifier := range selected {
		if group, err := ResolveGroup(groups, identifier); err == nil {
			selectedIDs[group.ID] = struct{}{}
		}
	}
	options := GroupOptions(groups)
	var completions []string
	for i, group := range groups {
		if _, ok := selectedIDs[group.ID]; ok {
			continue
		}
		if !strings.HasPrefix(group.Name, toComplete) && !strings.HasPrefix(group.ID, toComplete) {
			continue
		}
		identifier := group.Name
		if nameCounts[group.Name] > 1 {
			identifier = group.ID
		}
		completions = append(completions, fmt.Sprintf("%s\t%s", identifier, options[i].Key))
	}
	return completions
}

func groupBaseLabel(group api.Group) string {
	groupKind := "local"
	if IsOAuthGroup(group) {
		groupKind = "oauth"
	}
	metadata := groupKind
	if group.MemberCount != nil {
		metadata = fmt.Sprintf("%s, %d members", metadata, *group.MemberCount)
	}
	label := fmt.Sprintf("%s (%s)", group.Name, metadata)
	if strings.TrimSpace(group.Description) != "" {
		label += " - " + strings.TrimSpace(group.Description)
	}
	return label
}

// FilterUsersByRole returns only users with the given role.
func FilterUsersByRole(users []api.User, role string) []api.User {
	return Filter(users, func(u api.User) bool { return u.Role == role })
}
