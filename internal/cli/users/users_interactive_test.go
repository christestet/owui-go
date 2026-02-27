//go:build interactive
// +build interactive

package users

import (
	"bytes"
	"testing"
)

func TestCreateCommand_MissingName(t *testing.T) {
	server := newUsersServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	createCmd.SetOut(buf)

	// Set flags with missing name — but all others set to avoid interactive
	// Since name is empty, it will try to run wizard which will fail in test
	createCmd.Flags().Set("name", "")
	createCmd.Flags().Set("email", "test@example.com")
	createCmd.Flags().Set("password", "pass123")
	createCmd.Flags().Set("role", "user")
	defer func() {
		createCmd.Flags().Set("name", "")
		createCmd.Flags().Set("email", "")
		createCmd.Flags().Set("password", "")
		createCmd.Flags().Set("role", "")
	}()

	// The wizard will fail in a non-interactive test environment
	err := createCmd.RunE(createCmd, []string{})
	if err == nil {
		t.Fatal("expected error when interactive wizard can't run in test")
	}
}
