package groups

import (
	"encoding/json"
	"fmt"

	"github.com/christestet/owui-go/internal/api"
	"github.com/spf13/cobra"
)

type groupJSONRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func renderGroupPermissionJSON[T any](cmd *cobra.Command, group *api.Group, permission string, includePublic bool, read, write, public []T) error {
	out := struct {
		Group  groupJSONRef `json:"group"`
		Read   []T          `json:"read,omitempty"`
		Write  []T          `json:"write,omitempty"`
		Public []T          `json:"public,omitempty"`
	}{
		Group: groupJSONRef{ID: group.ID, Name: group.Name},
	}
	if permission == "all" || permission == "read" {
		out.Read = read
	}
	if permission == "all" || permission == "write" {
		out.Write = write
	}
	if includePublic {
		out.Public = public
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return err
}
