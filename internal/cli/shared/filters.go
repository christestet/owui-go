package shared

import (
	"fmt"
	"strings"

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

// FindGroupByName looks up a group by name from a group list.
func FindGroupByName(groups []api.Group, name string) (*api.Group, error) {
	for i := range groups {
		if groups[i].Name == name {
			return &groups[i], nil
		}
	}
	return nil, fmt.Errorf("group %q not found", name)
}

// FindUserByName looks up a user by name from a user list.
func FindUserByName(users []api.User, name string) (*api.User, error) {
	for i := range users {
		if users[i].Name == name {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("user %q not found", name)
}

// FilterUsersByRole returns only users with the given role.
func FilterUsersByRole(users []api.User, role string) []api.User {
	return Filter(users, func(u api.User) bool { return u.Role == role })
}
