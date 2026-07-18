package composer_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/model"
)

func TestResolveSelectsExactCompatibleOfficialRelease(t *testing.T) {
	t.Parallel()

	artifact := []byte("official Composer fixture")
	sum := sha256.Sum256(artifact)
	checksum := hex.EncodeToString(sum[:])
	var requestedMu sync.Mutex
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			requestedMu.Lock()
			requested = append(requested, request.URL.Path)
			requestedMu.Unlock()

			switch request.URL.Path {
			case "/versions":
				fmt.Fprint(writer, `{
					"stable": [
						{"path":"/download/2.9.5/composer.phar","version":"2.9.5","min-php":70205},
						{"path":"/download/2.8.9/composer.phar","version":"2.8.9","min-php":70205},
						{"path":"/download/2.8.8/composer.phar","version":"2.8.8","min-php":80500}
					]
				}`)
			case "/download/2.8.9/composer.phar.sha256sum":
				fmt.Fprintln(writer, checksum+"  composer.phar")
			default:
				http.NotFound(writer, request)
			}
		},
	))
	t.Cleanup(server.Close)

	manager := composer.NewManager(composer.ManagerOptions{
		CacheRoot: t.TempDir(),
		BaseURL:   server.URL,
		Client:    server.Client(),
	})
	release, err := manager.Resolve(context.Background(), composer.ResolveRequest{
		Constraint: "2.8.*",
		PHPVersion: "8.4.3",
	})
	if err != nil {
		t.Fatalf("resolve official Composer release: %v", err)
	}
	if release.Version != "2.8.9" {
		t.Fatalf("expected Composer 2.8.9, got %#v", release)
	}
	if release.URL != server.URL+"/download/2.8.9/composer.phar" {
		t.Fatalf("unexpected release URL %q", release.URL)
	}
	if release.SHA256 != "sha256:"+checksum {
		t.Fatalf("unexpected release checksum %q", release.SHA256)
	}

	requestedMu.Lock()
	defer requestedMu.Unlock()
	if len(requested) != 2 ||
		requested[0] != "/versions" ||
		requested[1] != "/download/2.8.9/composer.phar.sha256sum" {
		t.Fatalf("unexpected metadata requests %#v", requested)
	}
}

func TestAcquireVerifiesAndPromotesOfficialComposerAtomically(t *testing.T) {
	t.Parallel()

	artifact := []byte("#!/usr/bin/env php\nofficial Composer fixture\n")
	sum := sha256.Sum256(artifact)
	checksum := hex.EncodeToString(sum[:])
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/download/2.8.9/composer.phar" {
				http.NotFound(writer, request)

				return
			}
			_, _ = writer.Write(artifact)
		},
	))
	t.Cleanup(server.Close)

	cacheRoot := t.TempDir()
	manager := composer.NewManager(composer.ManagerOptions{
		CacheRoot: cacheRoot,
		BaseURL:   server.URL,
		Client:    server.Client(),
	})
	executable, err := manager.Acquire(
		context.Background(),
		composer.AcquireRequest{
			Release: composer.Release{
				Version:     "2.8.9",
				URL:         server.URL + "/download/2.8.9/composer.phar",
				SHA256:      "sha256:" + checksum,
				MetadataURL: server.URL + "/versions",
			},
		},
	)
	if err != nil {
		t.Fatalf("acquire official Composer executable: %v", err)
	}

	expectedPath := filepath.Join(
		cacheRoot,
		"composer",
		"artifacts",
		"sha256",
		checksum,
		"composer.phar",
	)
	if executable.Path != expectedPath ||
		executable.Version != "2.8.9" ||
		executable.Source != composer.SourceManaged ||
		executable.Identity != "sha256:"+checksum {
		t.Fatalf("unexpected managed Composer executable %#v", executable)
	}
	content, err := os.ReadFile(executable.Path)
	if err != nil {
		t.Fatalf("read promoted Composer executable: %v", err)
	}
	if string(content) != string(artifact) {
		t.Fatalf("unexpected promoted artifact %q", content)
	}
	info, err := os.Stat(executable.Path)
	if err != nil {
		t.Fatalf("inspect promoted Composer executable: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("promoted Composer is not executable, mode %s", info.Mode())
	}

	metadataContent, err := os.ReadFile(filepath.Join(
		filepath.Dir(executable.Path),
		"release.json",
	))
	if err != nil {
		t.Fatalf("read promoted release metadata: %v", err)
	}
	var metadata struct {
		URL        string `json:"url"`
		License    string `json:"license"`
		LicenseURL string `json:"license_url"`
	}
	if err := json.Unmarshal(metadataContent, &metadata); err != nil {
		t.Fatalf("decode promoted release metadata: %v", err)
	}
	if metadata.URL != server.URL+"/download/2.8.9/composer.phar" ||
		metadata.License != "MIT" ||
		metadata.LicenseURL == "" {
		t.Fatalf("unexpected release metadata %#v", metadata)
	}
}

func TestAcquireUsesOnlyCompleteVerifiedCacheEntriesOffline(t *testing.T) {
	t.Parallel()

	artifact := []byte("verified cached Composer")
	sum := sha256.Sum256(artifact)
	checksum := hex.EncodeToString(sum[:])
	var requestMu sync.Mutex
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			requestMu.Lock()
			requestCount++
			requestMu.Unlock()
			_, _ = writer.Write(artifact)
		},
	))
	t.Cleanup(server.Close)

	release := composer.Release{
		Version:     "2.8.9",
		URL:         server.URL + "/download/2.8.9/composer.phar",
		SHA256:      "sha256:" + checksum,
		MetadataURL: server.URL + "/versions",
	}
	manager := composer.NewManager(composer.ManagerOptions{
		CacheRoot: t.TempDir(),
		BaseURL:   server.URL,
		Client:    server.Client(),
	})
	first, err := manager.Acquire(context.Background(), composer.AcquireRequest{
		Release: release,
	})
	if err != nil {
		t.Fatalf("seed verified Composer cache: %v", err)
	}
	cached, err := manager.Acquire(context.Background(), composer.AcquireRequest{
		Release: release,
		Offline: true,
	})
	if err != nil {
		t.Fatalf("reuse verified Composer cache offline: %v", err)
	}
	if cached != first {
		t.Fatalf("cache hit changed executable identity\nfirst:  %#v\ncached: %#v", first, cached)
	}
	requestMu.Lock()
	if requestCount != 1 {
		t.Fatalf("offline cache hit made %d network requests", requestCount)
	}
	requestMu.Unlock()

	if err := os.Remove(filepath.Join(
		filepath.Dir(cached.Path),
		"release.json",
	)); err != nil {
		t.Fatalf("remove cache provenance fixture: %v", err)
	}
	_, err = manager.Acquire(context.Background(), composer.AcquireRequest{
		Release: release,
		Offline: true,
	})
	var commandError *model.Error
	if !errors.As(err, &commandError) ||
		commandError.Code != model.ErrorArtifact {
		t.Fatalf("expected missing provenance to invalidate cache, got %v", err)
	}
}

func TestAcquireRejectsChecksumFailureWithoutExecutableResidue(t *testing.T) {
	t.Parallel()

	expectedArtifact := []byte("expected official Composer")
	expectedSum := sha256.Sum256(expectedArtifact)
	expectedChecksum := hex.EncodeToString(expectedSum[:])
	receivedArtifact := []byte("tampered Composer")
	receivedSum := sha256.Sum256(receivedArtifact)
	receivedChecksum := hex.EncodeToString(receivedSum[:])
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write(receivedArtifact)
		},
	))
	t.Cleanup(server.Close)

	cacheRoot := t.TempDir()
	manager := composer.NewManager(composer.ManagerOptions{
		CacheRoot: cacheRoot,
		BaseURL:   server.URL,
		Client:    server.Client(),
	})
	_, err := manager.Acquire(context.Background(), composer.AcquireRequest{
		Release: composer.Release{
			Version:     "2.8.9",
			URL:         server.URL + "/download/2.8.9/composer.phar",
			SHA256:      "sha256:" + expectedChecksum,
			MetadataURL: server.URL + "/versions",
		},
	})
	var commandError *model.Error
	if !errors.As(err, &commandError) ||
		commandError.Code != model.ErrorArtifact {
		t.Fatalf("expected checksum artifact error, got %v", err)
	}
	details := make(map[string]string, len(commandError.Details))
	for _, detail := range commandError.Details {
		details[detail.Name] = detail.Value
	}
	if details["expected_sha256"] != "sha256:"+expectedChecksum ||
		details["actual_sha256"] != "sha256:"+receivedChecksum {
		t.Fatalf("unexpected checksum failure details %#v", commandError.Details)
	}

	artifactsRoot := filepath.Join(
		cacheRoot,
		"composer",
		"artifacts",
		"sha256",
	)
	entries, readErr := os.ReadDir(artifactsRoot)
	if readErr != nil {
		t.Fatalf("inspect Composer cache after checksum failure: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("checksum failure left cache residue %#v", entries)
	}
}

func TestAcquireRemovesInterruptedPartialDownload(t *testing.T) {
	t.Parallel()

	artifact := []byte("complete official Composer fixture")
	sum := sha256.Sum256(artifact)
	checksum := hex.EncodeToString(sum[:])
	partial := artifact[:11]
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Length", strconv.Itoa(len(artifact)))
			_, _ = writer.Write(partial)
		},
	))
	t.Cleanup(server.Close)

	cacheRoot := t.TempDir()
	manager := composer.NewManager(composer.ManagerOptions{
		CacheRoot: cacheRoot,
		BaseURL:   server.URL,
		Client:    server.Client(),
	})
	_, err := manager.Acquire(context.Background(), composer.AcquireRequest{
		Release: composer.Release{
			Version:     "2.8.9",
			URL:         server.URL + "/download/2.8.9/composer.phar",
			SHA256:      "sha256:" + checksum,
			MetadataURL: server.URL + "/versions",
		},
	})
	var commandError *model.Error
	if !errors.As(err, &commandError) ||
		commandError.Code != model.ErrorNetwork ||
		!commandError.Retryable {
		t.Fatalf("expected retryable interrupted download error, got %v", err)
	}
	if len(commandError.Details) != 1 ||
		commandError.Details[0].Name != "downloaded_bytes" ||
		commandError.Details[0].Value != strconv.Itoa(len(partial)) {
		t.Fatalf("unexpected interrupted download details %#v", commandError.Details)
	}

	artifactsRoot := filepath.Join(
		cacheRoot,
		"composer",
		"artifacts",
		"sha256",
	)
	entries, readErr := os.ReadDir(artifactsRoot)
	if readErr != nil {
		t.Fatalf("inspect Composer cache after interrupted download: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("interrupted download left cache residue %#v", entries)
	}
}

func TestAcquireOfflineMissFailsBeforeNetworkOrCacheMutation(t *testing.T) {
	t.Parallel()

	artifact := []byte("missing Composer fixture")
	sum := sha256.Sum256(artifact)
	checksum := hex.EncodeToString(sum[:])
	parent := t.TempDir()
	cacheRoot := filepath.Join(parent, "cache-not-created")
	baseURL := "https://getcomposer.example"
	manager := composer.NewManager(composer.ManagerOptions{
		CacheRoot: cacheRoot,
		BaseURL:   baseURL,
		Client: httpClientFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("offline cache miss attempted an HTTP request")

			return nil, errors.New("unreachable")
		}),
	})
	_, err := manager.Acquire(context.Background(), composer.AcquireRequest{
		Release: composer.Release{
			Version:     "2.8.9",
			URL:         baseURL + "/download/2.8.9/composer.phar",
			SHA256:      "sha256:" + checksum,
			MetadataURL: baseURL + "/versions",
		},
		Offline: true,
	})
	var commandError *model.Error
	if !errors.As(err, &commandError) ||
		commandError.Code != model.ErrorNetwork {
		t.Fatalf("expected offline cache miss network error, got %v", err)
	}
	expectedPath := filepath.Join(
		cacheRoot,
		"composer",
		"artifacts",
		"sha256",
		checksum,
		"composer.phar",
	)
	details := make(map[string]string, len(commandError.Details))
	for _, detail := range commandError.Details {
		details[detail.Name] = detail.Value
	}
	if details["identity"] != "sha256:"+checksum ||
		details["cache_path"] != expectedPath {
		t.Fatalf("unexpected offline miss details %#v", commandError.Details)
	}
	if _, statErr := os.Stat(cacheRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("offline miss mutated cache root, stat error %v", statErr)
	}
}

func TestResolveSelectsVerifiedManagedComposerFromCacheOffline(t *testing.T) {
	t.Parallel()

	artifact := []byte("offline managed Composer")
	sum := sha256.Sum256(artifact)
	checksum := hex.EncodeToString(sum[:])
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case "/versions":
				fmt.Fprint(writer, `{
					"stable": [
						{"path":"/download/2.8.9/composer.phar","version":"2.8.9","min-php":70205}
					]
				}`)
			case "/download/2.8.9/composer.phar.sha256sum":
				fmt.Fprintln(writer, checksum+"  composer.phar")
			case "/download/2.8.9/composer.phar":
				_, _ = writer.Write(artifact)
			default:
				http.NotFound(writer, request)
			}
		},
	))
	t.Cleanup(server.Close)

	cacheRoot := t.TempDir()
	manager := composer.NewManager(composer.ManagerOptions{
		CacheRoot: cacheRoot,
		BaseURL:   server.URL,
		Client:    server.Client(),
	})
	release, err := manager.Resolve(context.Background(), composer.ResolveRequest{
		Constraint: "2.8.*",
		PHPVersion: "8.4.3",
	})
	if err != nil {
		t.Fatalf("resolve Composer cache seed release: %v", err)
	}
	if _, err := manager.Acquire(
		context.Background(),
		composer.AcquireRequest{Release: release},
	); err != nil {
		t.Fatalf("seed verified Composer cache: %v", err)
	}

	offlineManager := composer.NewManager(composer.ManagerOptions{
		CacheRoot: cacheRoot,
		BaseURL:   server.URL,
		Client: httpClientFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("offline Composer resolution attempted an HTTP request")

			return nil, errors.New("unreachable")
		}),
	})
	cachedRelease, err := offlineManager.Resolve(
		context.Background(),
		composer.ResolveRequest{
			Constraint: "2.8.*",
			PHPVersion: "8.4.3",
			Offline:    true,
		},
	)
	if err != nil {
		t.Fatalf("resolve verified Composer cache offline: %v", err)
	}
	if cachedRelease != release {
		t.Fatalf(
			"offline resolution changed release\nonline:  %#v\noffline: %#v",
			release,
			cachedRelease,
		)
	}
}

type httpClientFunc func(*http.Request) (*http.Response, error)

func (function httpClientFunc) Do(
	request *http.Request,
) (*http.Response, error) {
	return function(request)
}
