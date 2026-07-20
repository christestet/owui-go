package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	selfupdate "github.com/creativeprojects/go-selfupdate/update"
)

const (
	repoSlug               = "christestet/owui-go"
	githubLatestReleaseURL = "https://api.github.com/repos/" + repoSlug + "/releases/latest"
	maxReleaseMetadataSize = 4 << 20
	maxChecksumFileSize    = 1 << 20
	maxArchiveSize         = 128 << 20
	maxBinarySize          = 128 << 20
)

// Release describes the archive and checksum file for an owui release.
type Release struct {
	version     string
	archiveName string
	archiveURL  string
	checksumURL string
	httpClient  *http.Client
}

// Version returns the release version without a leading "v".
func (r *Release) Version() string {
	return r.version
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type releaseClient struct {
	apiURL     string
	httpClient *http.Client
	goos       string
	goarch     string
}

// CheckLatest queries GitHub and returns (release, true, nil) if a newer version exists.
func CheckLatest(ctx context.Context, currentVersion string) (*Release, bool, error) {
	client := releaseClient{
		apiURL:     githubLatestReleaseURL,
		httpClient: http.DefaultClient,
		goos:       runtime.GOOS,
		goarch:     runtime.GOARCH,
	}
	return client.checkLatest(ctx, currentVersion)
}

func (c releaseClient) checkLatest(ctx context.Context, currentVersion string) (*Release, bool, error) {
	metadata, err := download(ctx, c.httpClient, c.apiURL, maxReleaseMetadataSize)
	if err != nil {
		return nil, false, fmt.Errorf("fetch latest release: %w", err)
	}

	var latest githubRelease
	if err := json.Unmarshal(metadata, &latest); err != nil {
		return nil, false, fmt.Errorf("decode latest release: %w", err)
	}

	latestVersion, err := semver.NewVersion(latest.TagName)
	if err != nil {
		return nil, false, fmt.Errorf("parse latest release version %q: %w", latest.TagName, err)
	}
	current, err := semver.NewVersion(currentVersion)
	if err != nil {
		return nil, false, fmt.Errorf("parse current version %q: %w", currentVersion, err)
	}

	release := &Release{
		version:    latestVersion.String(),
		httpClient: c.httpClient,
	}
	if !latestVersion.GreaterThan(current) {
		return release, false, nil
	}

	release.archiveName = fmt.Sprintf("owui_%s_%s_%s.tar.gz", latestVersion.String(), c.goos, c.goarch)
	for _, asset := range latest.Assets {
		switch asset.Name {
		case release.archiveName:
			release.archiveURL = asset.URL
		case "checksums.txt":
			release.checksumURL = asset.URL
		}
	}
	if release.archiveURL == "" {
		return nil, false, fmt.Errorf("release %s has no asset for %s/%s", release.version, c.goos, c.goarch)
	}
	if release.checksumURL == "" {
		return nil, false, fmt.Errorf("release %s has no checksums.txt asset", release.version)
	}

	return release, true, nil
}

// Apply downloads a release, verifies its archive checksum, and atomically replaces the running binary.
func Apply(ctx context.Context, release *Release) error {
	return applyRelease(ctx, release, "")
}

func applyRelease(ctx context.Context, release *Release, targetPath string) error {
	if release == nil {
		return fmt.Errorf("release is nil")
	}
	client := release.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	checksumFile, err := download(ctx, client, release.checksumURL, maxChecksumFileSize)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	expectedChecksum, err := checksumForAsset(checksumFile, release.archiveName)
	if err != nil {
		return err
	}

	archive, err := download(ctx, client, release.archiveURL, maxArchiveSize)
	if err != nil {
		return fmt.Errorf("download release archive: %w", err)
	}
	actualChecksum := sha256.Sum256(archive)
	if !bytes.Equal(expectedChecksum, actualChecksum[:]) {
		return fmt.Errorf("release archive checksum mismatch: expected %x, got %x", expectedChecksum, actualChecksum)
	}

	binary, err := extractBinary(archive)
	if err != nil {
		return fmt.Errorf("extract release archive: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}
	if err := selfupdate.Apply(bytes.NewReader(binary), selfupdate.Options{TargetPath: targetPath}); err != nil {
		return fmt.Errorf("replace executable: %w", err)
	}
	return nil
}

func download(ctx context.Context, client *http.Client, url string, limit int64) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("download URL is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "owui-self-update")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", url, err)
	}
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	closeErr := resp.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read %s: %w", url, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close response from %s: %w", url, closeErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request %s returned %s", url, resp.Status)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response from %s exceeds %d bytes", url, limit)
	}
	return data, nil
}

func checksumForAsset(checksumFile []byte, assetName string) ([]byte, error) {
	for line := range strings.SplitSeq(string(checksumFile), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != assetName {
			continue
		}
		checksum, err := hex.DecodeString(fields[0])
		if err != nil {
			return nil, fmt.Errorf("decode checksum for %s: %w", assetName, err)
		}
		if len(checksum) != sha256.Size {
			return nil, fmt.Errorf("checksum for %s has %d bytes, want %d", assetName, len(checksum), sha256.Size)
		}
		return checksum, nil
	}
	return nil, fmt.Errorf("checksums.txt has no entry for %s", assetName)
}

func extractBinary(archive []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open gzip stream: %w", err)
	}

	binary, readErr := readBinaryFromTar(tar.NewReader(gzipReader))
	closeErr := gzipReader.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close gzip stream: %w", closeErr)
	}
	return binary, nil
}

func readBinaryFromTar(reader *tar.Reader) ([]byte, error) {
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("archive does not contain owui binary")
		}
		if err != nil {
			return nil, fmt.Errorf("read tar archive: %w", err)
		}
		if path.Base(header.Name) != "owui" {
			continue
		}
		if !header.FileInfo().Mode().IsRegular() {
			return nil, fmt.Errorf("archive entry %q is not a regular file", header.Name)
		}
		binary, err := io.ReadAll(io.LimitReader(reader, maxBinarySize+1))
		if err != nil {
			return nil, fmt.Errorf("read owui binary: %w", err)
		}
		if len(binary) > maxBinarySize {
			return nil, fmt.Errorf("owui binary exceeds %d bytes", maxBinarySize)
		}
		return binary, nil
	}
}

// ShouldCheck returns true when 24h+ have elapsed since lastCheckStr (RFC3339).
// Empty or unparsable string means "never checked" and returns true.
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
