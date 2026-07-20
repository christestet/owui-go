package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReleaseClientCheckLatest(t *testing.T) {
	tests := []struct {
		name          string
		current       string
		tag           string
		assets        []string
		wantAvailable bool
		wantVersion   string
		wantErr       string
	}{
		{
			name:          "new release for current platform",
			current:       "1.1.0",
			tag:           "v1.2.0",
			assets:        []string{"owui_1.2.0_linux_amd64.tar.gz", "checksums.txt"},
			wantAvailable: true,
			wantVersion:   "1.2.0",
		},
		{
			name:        "already current",
			current:     "1.2.0",
			tag:         "v1.2.0",
			wantVersion: "1.2.0",
		},
		{
			name:    "missing platform archive",
			current: "1.1.0",
			tag:     "v1.2.0",
			assets:  []string{"checksums.txt"},
			wantErr: "has no asset for linux/amd64",
		},
		{
			name:    "invalid release tag",
			current: "1.1.0",
			tag:     "latest",
			wantErr: "parse latest release version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				assets := make([]string, 0, len(tt.assets))
				for _, name := range tt.assets {
					assets = append(assets, fmt.Sprintf(`{"name":%q,"browser_download_url":%q}`, name, "https://example.test/"+name))
				}
				body := fmt.Sprintf(`{"tag_name":%q,"assets":[%s]}`, tt.tag, strings.Join(assets, ","))
				return testHTTPResponse(req, http.StatusOK, []byte(body)), nil
			})}

			client := releaseClient{
				apiURL:     "https://example.test/latest",
				httpClient: httpClient,
				goos:       "linux",
				goarch:     "amd64",
			}
			release, available, err := client.checkLatest(context.Background(), tt.current)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("checkLatest() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("checkLatest() error = %v", err)
			}
			if available != tt.wantAvailable {
				t.Errorf("checkLatest() available = %v, want %v", available, tt.wantAvailable)
			}
			if release.Version() != tt.wantVersion {
				t.Errorf("release.Version() = %q, want %q", release.Version(), tt.wantVersion)
			}
		})
	}
}

func TestChecksumForAsset(t *testing.T) {
	validChecksum := strings.Repeat("ab", sha256.Size)
	tests := []struct {
		name    string
		file    string
		asset   string
		wantErr string
	}{
		{
			name:  "valid checksum",
			file:  validChecksum + "  owui_1.2.0_linux_amd64.tar.gz\n",
			asset: "owui_1.2.0_linux_amd64.tar.gz",
		},
		{
			name:  "binary marker",
			file:  validChecksum + " *owui_1.2.0_linux_amd64.tar.gz\n",
			asset: "owui_1.2.0_linux_amd64.tar.gz",
		},
		{
			name:    "missing asset",
			file:    validChecksum + "  other.tar.gz\n",
			asset:   "owui_1.2.0_linux_amd64.tar.gz",
			wantErr: "has no entry",
		},
		{
			name:    "invalid checksum length",
			file:    "abcd  owui.tar.gz\n",
			asset:   "owui.tar.gz",
			wantErr: "has 2 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum, err := checksumForAsset([]byte(tt.file), tt.asset)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("checksumForAsset() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("checksumForAsset() error = %v", err)
			}
			if len(checksum) != sha256.Size {
				t.Errorf("checksum length = %d, want %d", len(checksum), sha256.Size)
			}
		})
	}
}

func TestApplyRelease(t *testing.T) {
	const archiveName = "owui_1.2.0_linux_amd64.tar.gz"
	archive := makeTestArchive(t, []byte("new binary"))
	checksum := sha256.Sum256(archive)

	tests := []struct {
		name            string
		checksumFile    string
		wantErr         string
		wantTargetValue string
	}{
		{
			name:            "verified update",
			checksumFile:    fmt.Sprintf("%x  %s\n", checksum, archiveName),
			wantTargetValue: "new binary",
		},
		{
			name:            "checksum mismatch",
			checksumFile:    strings.Repeat("00", sha256.Size) + "  " + archiveName + "\n",
			wantErr:         "checksum mismatch",
			wantTargetValue: "old binary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				var data []byte
				switch req.URL.Path {
				case "/checksums.txt":
					data = []byte(tt.checksumFile)
				case "/" + archiveName:
					data = archive
				default:
					return testHTTPResponse(req, http.StatusNotFound, nil), nil
				}
				return testHTTPResponse(req, http.StatusOK, data), nil
			})}

			target := filepath.Join(t.TempDir(), "owui")
			if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
				t.Fatalf("write target: %v", err)
			}
			release := &Release{
				version:     "1.2.0",
				archiveName: archiveName,
				archiveURL:  "https://example.test/" + archiveName,
				checksumURL: "https://example.test/checksums.txt",
				httpClient:  httpClient,
			}

			err := applyRelease(context.Background(), release, target)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("applyRelease() error = %v, want error containing %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("applyRelease() error = %v", err)
			}

			got, err := os.ReadFile(target)
			if err != nil {
				t.Fatalf("read target: %v", err)
			}
			if string(got) != tt.wantTargetValue {
				t.Errorf("target content = %q, want %q", got, tt.wantTargetValue)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPResponse(req *http.Request, statusCode int, data []byte) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Body:       io.NopCloser(bytes.NewReader(data)),
		Header:     make(http.Header),
		Request:    req,
	}
}

func makeTestArchive(t *testing.T, binary []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	header := &tar.Header{
		Name: "owui",
		Mode: 0o755,
		Size: int64(len(binary)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tarWriter.Write(binary); err != nil {
		t.Fatalf("write tar body: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buffer.Bytes()
}

func TestShouldCheck(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", true},
		{"invalid", "not-a-time", true},
		{"25h ago", time.Now().Add(-25 * time.Hour).UTC().Format(time.RFC3339), true},
		{"23h ago", time.Now().Add(-23 * time.Hour).UTC().Format(time.RFC3339), false},
		{"just now", time.Now().UTC().Format(time.RFC3339), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldCheck(tt.in); got != tt.want {
				t.Errorf("ShouldCheck(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
