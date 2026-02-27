package updater

import (
	"context"
	"time"

	"github.com/creativeprojects/go-selfupdate"
)

const repoSlug = "christestet/owui-go"

// CheckLatest queries GitHub and returns (release, true, nil) if newer version exists.
func CheckLatest(ctx context.Context, currentVersion string) (*selfupdate.Release, bool, error) {
	u, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return nil, false, err
	}
	release, found, err := u.DetectLatest(ctx, selfupdate.ParseSlug(repoSlug))
	if err != nil {
		return nil, false, err
	}
	if !found || !release.GreaterThan(currentVersion) {
		return release, false, nil
	}
	return release, true, nil
}

// Apply downloads the release and atomically replaces the running binary.
func Apply(ctx context.Context, release *selfupdate.Release) error {
	u, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return err
	}
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return err
	}
	return u.UpdateTo(ctx, release, exe)
}

// ShouldCheck returns true when 24h+ have elapsed since lastCheckStr (RFC3339).
// Empty or unparsable string means "never checked" → returns true.
func ShouldCheck(lastCheckStr string) bool {
	if lastCheckStr == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, lastCheckStr)
	if err != nil {
		return true
	}
	return time.Since(t) >= 24*time.Hour
}
