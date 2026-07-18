package composer

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/elefantephp/elefante/internal/constraints"
	"github.com/elefantephp/elefante/internal/model"
)

const (
	officialBaseURL      = "https://getcomposer.org"
	officialLicenseURL   = "https://github.com/composer/composer/blob/main/LICENSE"
	maximumMetadataBytes = 1 << 20
	maximumArtifactBytes = 64 << 20
	releaseSchemaVersion = "elefante.composer-release/v1"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type ManagerOptions struct {
	CacheRoot string
	BaseURL   string
	Client    HTTPClient
}

type Manager struct {
	cacheRoot string
	baseURL   *url.URL
	client    HTTPClient
}

type ResolveRequest struct {
	Constraint string
	PHPVersion string
	Offline    bool
}

type Release struct {
	Version     string `json:"version"`
	URL         string `json:"url"`
	SHA256      string `json:"sha256"`
	MetadataURL string `json:"metadata_url"`
	MinimumPHP  int    `json:"minimum_php,omitempty"`
}

type AcquireRequest struct {
	Release Release
	Offline bool
}

type Executable struct {
	Version  string `json:"version"`
	Source   string `json:"source"`
	Path     string `json:"path"`
	Identity string `json:"identity"`
	SHA256   string `json:"sha256"`
}

type releaseRecord struct {
	SchemaVersion string  `json:"schema_version"`
	Release       Release `json:"release"`
	URL           string  `json:"url"`
	License       string  `json:"license"`
	LicenseURL    string  `json:"license_url"`
}

type versionsDocument struct {
	Stable []releaseDocument `json:"stable"`
}

type releaseDocument struct {
	Path       string `json:"path"`
	Version    string `json:"version"`
	MinimumPHP int    `json:"min-php"`
}

func NewManager(options ManagerOptions) Manager {
	baseURL := strings.TrimRight(strings.TrimSpace(options.BaseURL), "/")
	if baseURL == "" {
		baseURL = officialBaseURL
	}
	parsedBaseURL, _ := url.Parse(baseURL)
	client := options.Client
	if client == nil {
		client = http.DefaultClient
	}

	return Manager{
		cacheRoot: options.CacheRoot,
		baseURL:   parsedBaseURL,
		client:    client,
	}
}

func (manager Manager) Resolve(
	ctx context.Context,
	request ResolveRequest,
) (Release, error) {
	if request.Offline {
		return manager.resolveCached(ctx, request)
	}
	if manager.baseURL == nil ||
		manager.baseURL.Scheme == "" ||
		manager.baseURL.Host == "" {
		return Release{}, model.NewError(
			model.ErrorArtifact,
			"The Composer distribution source is invalid.",
		)
	}

	metadataURL := manager.resolveURL("/versions")
	var document versionsDocument
	if err := manager.getJSON(ctx, metadataURL, &document); err != nil {
		return Release{}, err
	}
	candidates, err := compatibleReleases(document.Stable, request)
	if err != nil {
		return Release{}, err
	}
	if len(candidates) == 0 {
		return Release{}, model.NewError(
			model.ErrorRequirements,
			"Official Composer metadata has no compatible stable release.",
		).WithHint("Adjust the committed Composer policy or selected PHP runtime.")
	}
	sort.Slice(candidates, func(left int, right int) bool {
		leftVersion, leftErr := constraints.NormalizeVersion(
			candidates[left].Version,
		)
		rightVersion, rightErr := constraints.NormalizeVersion(
			candidates[right].Version,
		)
		if leftErr == nil && rightErr == nil {
			return leftVersion.Compare(rightVersion) > 0
		}

		return candidates[left].Version > candidates[right].Version
	})

	selected := candidates[0]
	downloadURL, err := manager.distributionURL(selected.Path)
	if err != nil {
		return Release{}, err
	}
	checksumURL := downloadURL + ".sha256sum"
	checksum, err := manager.getChecksum(ctx, checksumURL)
	if err != nil {
		return Release{}, err
	}

	return Release{
		Version:     selected.Version,
		URL:         downloadURL,
		SHA256:      "sha256:" + checksum,
		MetadataURL: metadataURL,
		MinimumPHP:  selected.MinimumPHP,
	}, nil
}

func (manager Manager) resolveCached(
	ctx context.Context,
	request ResolveRequest,
) (Release, error) {
	artifactsRoot := filepath.Join(
		manager.cacheRoot,
		"composer",
		"artifacts",
		"sha256",
	)
	entries, err := os.ReadDir(artifactsRoot)
	if os.IsNotExist(err) {
		return Release{}, offlineReleaseMiss(artifactsRoot)
	}
	if err != nil {
		return Release{}, artifactError(
			"Could not inspect the managed Composer cache.",
			err,
		)
	}

	var candidates []Release
	for _, entry := range entries {
		if !entry.IsDir() || !validSHA256(entry.Name()) {
			continue
		}
		select {
		case <-ctx.Done():
			return Release{}, model.WrapError(
				model.ErrorNetwork,
				"Offline Composer cache inspection was canceled.",
				ctx.Err(),
			)
		default:
		}
		recordPath := filepath.Join(
			artifactsRoot,
			entry.Name(),
			"release.json",
		)
		content, err := os.ReadFile(recordPath)
		if err != nil {
			return Release{}, artifactError(
				"Could not read cached Composer release metadata.",
				err,
			)
		}
		var record releaseRecord
		if err := json.Unmarshal(content, &record); err != nil {
			return Release{}, artifactError(
				"Cached Composer release metadata is invalid.",
				err,
			)
		}
		if err := validateReleaseRecord(recordPath, record.Release); err != nil {
			return Release{}, err
		}
		checksum, err := manager.validateRelease(record.Release)
		if err != nil {
			return Release{}, err
		}
		if checksum != entry.Name() {
			return Release{}, model.NewError(
				model.ErrorArtifact,
				"The cached Composer release directory does not match its checksum.",
			)
		}
		_, exists, err := cachedExecutable(
			filepath.Join(filepath.Dir(recordPath), "composer.phar"),
			record.Release,
			checksum,
		)
		if err != nil {
			return Release{}, err
		}
		if !exists {
			continue
		}
		compatible, err := cachedReleaseCompatible(record.Release, request)
		if err != nil {
			return Release{}, err
		}
		if compatible {
			candidates = append(candidates, record.Release)
		}
	}
	if len(candidates) == 0 {
		return Release{}, offlineReleaseMiss(artifactsRoot)
	}
	sort.Slice(candidates, func(left int, right int) bool {
		leftVersion, _ := constraints.NormalizeVersion(
			candidates[left].Version,
		)
		rightVersion, _ := constraints.NormalizeVersion(
			candidates[right].Version,
		)

		return leftVersion.Compare(rightVersion) > 0
	})

	return candidates[0], nil
}

func cachedReleaseCompatible(
	release Release,
	request ResolveRequest,
) (bool, error) {
	if release.MinimumPHP <= 0 {
		return false, nil
	}
	runtimeVersion, err := constraints.NormalizeVersion(request.PHPVersion)
	if err != nil {
		return false, model.WrapError(
			model.ErrorRequirements,
			"The selected PHP runtime version is invalid.",
			err,
		)
	}
	minimumPHP, err := constraints.NormalizeVersion(
		phpVersionFromID(release.MinimumPHP),
	)
	if err != nil {
		return false, model.WrapError(
			model.ErrorArtifact,
			"Cached Composer metadata contains an invalid PHP minimum.",
			err,
		)
	}
	if runtimeVersion.Compare(minimumPHP) < 0 {
		return false, nil
	}
	if strings.TrimSpace(request.Constraint) == "" {
		return true, nil
	}
	matches, err := constraints.Satisfies(
		release.Version,
		request.Constraint,
	)
	if err != nil {
		return false, model.WrapError(
			model.ErrorRequirements,
			"The committed Composer constraint is invalid.",
			err,
		)
	}

	return matches, nil
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)

	return err == nil && len(decoded) == 32
}

func offlineReleaseMiss(cachePath string) *model.Error {
	commandError := model.NewError(
		model.ErrorNetwork,
		"Offline mode has no compatible verified managed Composer release.",
	).WithHint("Acquire a compatible official Composer release while online, then retry.")
	commandError.Details = []model.ErrorDetail{
		{Name: "cache_path", Value: cachePath},
	}

	return commandError
}

func (manager Manager) Observation(
	release Release,
) (model.ComposerObservation, error) {
	checksum, err := manager.validateRelease(release)
	if err != nil {
		return model.ComposerObservation{}, err
	}
	targetPath := manager.artifactPath(checksum)
	_, cached, err := cachedExecutable(targetPath, release, checksum)
	if err != nil {
		return model.ComposerObservation{}, err
	}

	return model.ComposerObservation{
		Version:         release.Version,
		Source:          SourceManaged,
		Path:            targetPath,
		Identity:        "sha256:" + checksum,
		SHA256:          "sha256:" + checksum,
		DistributionURL: release.URL,
		MetadataURL:     release.MetadataURL,
		Cached:          cached,
		Reference: model.SourceReference{
			Path: release.URL,
			Kind: "composer_distribution",
		},
	}, nil
}

func (manager Manager) Acquire(
	ctx context.Context,
	request AcquireRequest,
) (Executable, error) {
	checksum, err := manager.validateRelease(request.Release)
	if err != nil {
		return Executable{}, err
	}
	targetPath := manager.artifactPath(checksum)
	targetDirectory := filepath.Dir(targetPath)
	if executable, exists, err := cachedExecutable(
		targetPath,
		request.Release,
		checksum,
	); err != nil || exists {
		return executable, err
	}
	if request.Offline {
		commandError := model.NewError(
			model.ErrorNetwork,
			"Offline mode requires the selected Composer artifact in the verified cache.",
		).WithHint("Acquire the approved Composer release while online, then retry.")
		commandError.Details = []model.ErrorDetail{
			{Name: "identity", Value: "sha256:" + checksum},
			{Name: "cache_path", Value: targetPath},
		}

		return Executable{}, commandError
	}

	artifactsRoot := filepath.Dir(targetDirectory)
	if err := os.MkdirAll(artifactsRoot, 0o700); err != nil {
		return Executable{}, artifactError(
			"Could not create the managed Composer cache.",
			err,
		)
	}
	if err := os.Chmod(artifactsRoot, 0o700); err != nil {
		return Executable{}, artifactError(
			"Could not secure the managed Composer cache.",
			err,
		)
	}
	stagingDirectory, err := os.MkdirTemp(
		artifactsRoot,
		".composer-partial-*",
	)
	if err != nil {
		return Executable{}, artifactError(
			"Could not create a temporary Composer download.",
			err,
		)
	}
	removeStaging := true
	defer func() {
		if removeStaging {
			_ = os.RemoveAll(stagingDirectory)
		}
	}()
	if err := os.Chmod(stagingDirectory, 0o700); err != nil {
		return Executable{}, artifactError(
			"Could not secure the temporary Composer download.",
			err,
		)
	}

	stagingPath := filepath.Join(stagingDirectory, "composer.phar")
	if err := manager.downloadAndVerify(
		ctx,
		request.Release,
		stagingPath,
		checksum,
	); err != nil {
		return Executable{}, err
	}
	if err := writeReleaseRecord(
		filepath.Join(stagingDirectory, "release.json"),
		request.Release,
	); err != nil {
		return Executable{}, err
	}
	if err := syncDirectory(stagingDirectory); err != nil {
		return Executable{}, artifactError(
			"Could not flush the verified Composer artifact.",
			err,
		)
	}
	if err := os.Rename(stagingDirectory, targetDirectory); err != nil {
		if executable, exists, cacheErr := cachedExecutable(
			targetPath,
			request.Release,
			checksum,
		); cacheErr != nil || exists {
			return executable, cacheErr
		}

		return Executable{}, artifactError(
			"Could not atomically promote the verified Composer artifact.",
			err,
		)
	}
	removeStaging = false
	if err := syncDirectory(artifactsRoot); err != nil {
		return Executable{}, artifactError(
			"Could not flush the managed Composer cache.",
			err,
		)
	}

	return managedExecutable(targetPath, request.Release, checksum), nil
}

func (manager Manager) artifactPath(checksum string) string {
	return filepath.Join(
		manager.cacheRoot,
		"composer",
		"artifacts",
		"sha256",
		checksum,
		"composer.phar",
	)
}

func (manager Manager) downloadAndVerify(
	ctx context.Context,
	release Release,
	targetPath string,
	expectedChecksum string,
) error {
	response, err := manager.get(ctx, release.URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	target, err := os.OpenFile(
		targetPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if err != nil {
		return artifactError(
			"Could not create the temporary Composer artifact.",
			err,
		)
	}
	removeTarget := true
	defer func() {
		_ = target.Close()
		if removeTarget {
			_ = os.Remove(targetPath)
		}
	}()

	hasher := sha256.New()
	limited := &io.LimitedReader{
		R: response.Body,
		N: maximumArtifactBytes + 1,
	}
	written, err := io.Copy(io.MultiWriter(target, hasher), limited)
	if err != nil {
		commandError := model.WrapError(
			model.ErrorNetwork,
			"The Composer download was interrupted.",
			err,
		).WithRetryable(true)
		commandError.Details = []model.ErrorDetail{
			{Name: "downloaded_bytes", Value: fmt.Sprintf("%d", written)},
		}

		return commandError
	}
	if written > maximumArtifactBytes {
		return model.NewError(
			model.ErrorArtifact,
			"The Composer artifact exceeds the supported size.",
		)
	}
	if err := target.Sync(); err != nil {
		return artifactError(
			"Could not flush the temporary Composer artifact.",
			err,
		)
	}
	if err := target.Close(); err != nil {
		return artifactError(
			"Could not close the temporary Composer artifact.",
			err,
		)
	}
	actualChecksum := hasher.Sum(nil)
	expectedBytes, _ := hex.DecodeString(expectedChecksum)
	if subtle.ConstantTimeCompare(actualChecksum, expectedBytes) != 1 {
		commandError := model.NewError(
			model.ErrorArtifact,
			"The downloaded Composer artifact failed checksum verification.",
		).WithHint("Discard the unverified download and retry from the official distribution.")
		commandError.Details = []model.ErrorDetail{
			{Name: "expected_sha256", Value: "sha256:" + expectedChecksum},
			{
				Name:  "actual_sha256",
				Value: "sha256:" + hex.EncodeToString(actualChecksum),
			},
		}

		return commandError
	}
	if err := os.Chmod(targetPath, 0o700); err != nil {
		return artifactError(
			"Could not mark the verified Composer artifact executable.",
			err,
		)
	}
	removeTarget = false

	return nil
}

func (manager Manager) validateRelease(release Release) (string, error) {
	if _, err := constraints.NormalizeVersion(release.Version); err != nil {
		return "", model.WrapError(
			model.ErrorArtifact,
			"The selected Composer release version is invalid.",
			err,
		)
	}
	checksum := strings.TrimPrefix(
		strings.ToLower(strings.TrimSpace(release.SHA256)),
		"sha256:",
	)
	decoded, err := hex.DecodeString(checksum)
	if err != nil || len(decoded) != 32 {
		return "", model.NewError(
			model.ErrorArtifact,
			"The selected Composer release checksum is invalid.",
		)
	}
	for _, address := range []string{release.URL, release.MetadataURL} {
		parsed, err := url.Parse(address)
		if err != nil ||
			parsed.Scheme != manager.baseURL.Scheme ||
			parsed.Host != manager.baseURL.Host {
			return "", model.NewError(
				model.ErrorArtifact,
				"The selected Composer release is outside its approved distribution.",
			)
		}
	}
	if strings.TrimSpace(manager.cacheRoot) == "" {
		return "", model.NewError(
			model.ErrorArtifact,
			"The managed Composer cache root is not configured.",
		)
	}

	return checksum, nil
}

func cachedExecutable(
	targetPath string,
	release Release,
	expectedChecksum string,
) (Executable, bool, error) {
	info, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		return Executable{}, false, nil
	}
	if err != nil {
		return Executable{}, false, artifactError(
			"Could not inspect the cached Composer artifact.",
			err,
		)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return Executable{}, false, model.NewError(
			model.ErrorArtifact,
			"The cached Composer artifact is not a verified executable file.",
		)
	}
	if err := validateReleaseRecord(
		filepath.Join(filepath.Dir(targetPath), "release.json"),
		release,
	); err != nil {
		return Executable{}, false, err
	}
	file, err := os.Open(targetPath)
	if err != nil {
		return Executable{}, false, artifactError(
			"Could not open the cached Composer artifact.",
			err,
		)
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, io.LimitReader(file, maximumArtifactBytes+1)); err != nil {
		return Executable{}, false, artifactError(
			"Could not verify the cached Composer artifact.",
			err,
		)
	}
	expectedBytes, _ := hex.DecodeString(expectedChecksum)
	if subtle.ConstantTimeCompare(hasher.Sum(nil), expectedBytes) != 1 {
		return Executable{}, false, model.NewError(
			model.ErrorArtifact,
			"The cached Composer artifact failed checksum verification.",
		)
	}

	return managedExecutable(targetPath, release, expectedChecksum), true, nil
}

func validateReleaseRecord(path string, release Release) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return artifactError(
			"Could not read the cached Composer release metadata.",
			err,
		)
	}
	if len(content) > maximumMetadataBytes {
		return model.NewError(
			model.ErrorArtifact,
			"The cached Composer release metadata exceeds the supported size.",
		)
	}
	var record releaseRecord
	if err := json.Unmarshal(content, &record); err != nil {
		return artifactError(
			"The cached Composer release metadata is invalid.",
			err,
		)
	}
	if record.SchemaVersion != releaseSchemaVersion ||
		record.Release != release ||
		record.URL != release.URL ||
		record.License != "MIT" ||
		record.LicenseURL != officialLicenseURL {
		return model.NewError(
			model.ErrorArtifact,
			"The cached Composer release metadata does not match the selected release.",
		)
	}

	return nil
}

func managedExecutable(
	path string,
	release Release,
	checksum string,
) Executable {
	identity := "sha256:" + checksum

	return Executable{
		Version:  release.Version,
		Source:   SourceManaged,
		Path:     path,
		Identity: identity,
		SHA256:   identity,
	}
}

func writeReleaseRecord(path string, release Release) error {
	encoded, err := json.MarshalIndent(releaseRecord{
		SchemaVersion: releaseSchemaVersion,
		Release:       release,
		URL:           release.URL,
		License:       "MIT",
		LicenseURL:    officialLicenseURL,
	}, "", "  ")
	if err != nil {
		return artifactError("Could not encode Composer release metadata.", err)
	}
	encoded = append(encoded, '\n')
	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if err != nil {
		return artifactError("Could not create Composer release metadata.", err)
	}
	if _, err := file.Write(encoded); err != nil {
		_ = file.Close()

		return artifactError("Could not write Composer release metadata.", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()

		return artifactError("Could not flush Composer release metadata.", err)
	}
	if err := file.Close(); err != nil {
		return artifactError("Could not close Composer release metadata.", err)
	}

	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()

	return directory.Sync()
}

func artifactError(message string, cause error) *model.Error {
	return model.WrapError(model.ErrorArtifact, message, cause)
}

func compatibleReleases(
	releases []releaseDocument,
	request ResolveRequest,
) ([]releaseDocument, error) {
	runtimeVersion, err := constraints.NormalizeVersion(request.PHPVersion)
	if err != nil {
		return nil, model.WrapError(
			model.ErrorRequirements,
			"The selected PHP runtime version is invalid.",
			err,
		)
	}

	compatible := make([]releaseDocument, 0, len(releases))
	for _, release := range releases {
		matches := true
		if strings.TrimSpace(request.Constraint) != "" {
			matches, err = constraints.Satisfies(
				release.Version,
				request.Constraint,
			)
			if err != nil {
				return nil, model.WrapError(
					model.ErrorRequirements,
					"The committed Composer constraint is invalid.",
					err,
				)
			}
		}
		if !matches {
			continue
		}
		minimumPHP, err := constraints.NormalizeVersion(
			phpVersionFromID(release.MinimumPHP),
		)
		if err != nil {
			return nil, model.WrapError(
				model.ErrorArtifact,
				"Official Composer metadata contains an invalid PHP minimum.",
				err,
			)
		}
		if runtimeVersion.Compare(minimumPHP) >= 0 {
			compatible = append(compatible, release)
		}
	}

	return compatible, nil
}

func phpVersionFromID(identifier int) string {
	return fmt.Sprintf(
		"%d.%d.%d",
		identifier/10000,
		(identifier/100)%100,
		identifier%100,
	)
}

func (manager Manager) getJSON(
	ctx context.Context,
	address string,
	target any,
) error {
	response, err := manager.get(ctx, address)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	decoder := json.NewDecoder(io.LimitReader(
		response.Body,
		maximumMetadataBytes,
	))
	if err := decoder.Decode(target); err != nil {
		return model.WrapError(
			model.ErrorArtifact,
			"Official Composer release metadata is invalid.",
			err,
		)
	}

	return nil
}

func (manager Manager) getChecksum(
	ctx context.Context,
	address string,
) (string, error) {
	response, err := manager.get(ctx, address)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	content, err := io.ReadAll(io.LimitReader(response.Body, 256))
	if err != nil {
		return "", model.WrapError(
			model.ErrorNetwork,
			"Could not read the official Composer checksum.",
			err,
		)
	}
	fields := strings.Fields(string(content))
	if len(fields) != 2 || fields[1] != "composer.phar" {
		return "", model.NewError(
			model.ErrorArtifact,
			"The official Composer checksum record is invalid.",
		)
	}
	checksum := fields[0]
	decoded, err := hex.DecodeString(checksum)
	if err != nil || len(decoded) != 32 {
		return "", model.NewError(
			model.ErrorArtifact,
			"The official Composer checksum is invalid.",
		)
	}

	return strings.ToLower(checksum), nil
}

func (manager Manager) get(
	ctx context.Context,
	address string,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, model.WrapError(
			model.ErrorInternal,
			"Could not create the Composer metadata request.",
			err,
		)
	}
	request.Header.Set("Accept", "application/json, text/plain")
	request.Header.Set("User-Agent", "Elefante")
	response, err := manager.client.Do(request)
	if err != nil {
		return nil, model.WrapError(
			model.ErrorNetwork,
			"Could not reach the official Composer distribution.",
			err,
		).WithRetryable(true)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		commandError := model.NewError(
			model.ErrorNetwork,
			"The official Composer distribution returned an unexpected status.",
		).WithRetryable(response.StatusCode >= http.StatusInternalServerError)
		commandError.Details = []model.ErrorDetail{
			{Name: "status", Value: response.Status},
		}

		return nil, commandError
	}

	return response, nil
}

func (manager Manager) distributionURL(path string) (string, error) {
	reference, err := url.Parse(path)
	if err != nil {
		return "", model.WrapError(
			model.ErrorArtifact,
			"Official Composer metadata contains an invalid release path.",
			err,
		)
	}
	resolved := manager.baseURL.ResolveReference(reference)
	if resolved.Scheme != manager.baseURL.Scheme ||
		resolved.Host != manager.baseURL.Host {
		return "", model.NewError(
			model.ErrorArtifact,
			"Official Composer metadata points outside its approved distribution.",
		)
	}

	return resolved.String(), nil
}

func (manager Manager) resolveURL(path string) string {
	reference, _ := url.Parse(path)

	return manager.baseURL.ResolveReference(reference).String()
}
