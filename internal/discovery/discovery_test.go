package discovery_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/discovery"
	"github.com/elefantephp/elefante/internal/model"
)

func TestDiscoverSelectsNearestComposerRootFromDescendant(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")
	resolvedProjectRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}

	startPath := filepath.Join(projectRoot, "app", "Services")
	if err := os.MkdirAll(startPath, 0o755); err != nil {
		t.Fatalf("create descendant: %v", err)
	}

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: startPath,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	if facts.Identity.ComposerRoot != resolvedProjectRoot {
		t.Errorf("expected Composer root %q, got %q", resolvedProjectRoot, facts.Identity.ComposerRoot)
	}
	if facts.Identity.ApplicationRoot != resolvedProjectRoot {
		t.Errorf("expected application root %q, got %q", resolvedProjectRoot, facts.Identity.ApplicationRoot)
	}
	if facts.Identity.WorkspaceRoot != resolvedProjectRoot {
		t.Errorf("expected workspace root %q, got %q", resolvedProjectRoot, facts.Identity.WorkspaceRoot)
	}
	if facts.Identity.RepositoryRoot != "" {
		t.Errorf("expected no repository root, got %q", facts.Identity.RepositoryRoot)
	}
}

func TestDiscoverReadsGitRepositoryMetadataWithoutRemote(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")
	runGit(t, projectRoot, "init", "-b", "main")
	runGit(t, projectRoot, "config", "user.name", "Elefante Tests")
	runGit(t, projectRoot, "config", "user.email", "tests@elefante.local")
	runGit(t, projectRoot, "add", "composer.json")
	runGit(t, projectRoot, "commit", "-m", "Initial fixture")

	resolvedProjectRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	expectedHead := runGit(t, projectRoot, "rev-parse", "HEAD")

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	if facts.Identity.RepositoryRoot != resolvedProjectRoot {
		t.Errorf("expected repository root %q, got %q", resolvedProjectRoot, facts.Identity.RepositoryRoot)
	}
	if facts.Identity.WorkspaceRoot != resolvedProjectRoot {
		t.Errorf("expected workspace root %q, got %q", resolvedProjectRoot, facts.Identity.WorkspaceRoot)
	}
	if facts.Identity.GitCommonDir != filepath.Join(resolvedProjectRoot, ".git") {
		t.Errorf("expected Git common directory under repository root, got %q", facts.Identity.GitCommonDir)
	}
	if facts.Identity.Branch != "main" {
		t.Errorf("expected branch main, got %q", facts.Identity.Branch)
	}
	if facts.Identity.HeadCommit != expectedHead {
		t.Errorf("expected HEAD %q, got %q", expectedHead, facts.Identity.HeadCommit)
	}
	if !strings.HasPrefix(facts.Identity.IdentityKey, "sha256:") {
		t.Errorf("expected sha256 identity key, got %q", facts.Identity.IdentityKey)
	}
}

func TestDiscoverAssignsDistinctIdentityToLinkedWorktree(t *testing.T) {
	repositoryRoot := t.TempDir()
	writeFile(t, filepath.Join(repositoryRoot, "composer.json"), "{}\n")
	runGit(t, repositoryRoot, "init", "-b", "main")
	runGit(t, repositoryRoot, "config", "user.name", "Elefante Tests")
	runGit(t, repositoryRoot, "config", "user.email", "tests@elefante.local")
	runGit(t, repositoryRoot, "add", "composer.json")
	runGit(t, repositoryRoot, "commit", "-m", "Initial fixture")

	worktreeRoot := filepath.Join(t.TempDir(), "feature")
	runGit(t, repositoryRoot, "worktree", "add", "-b", "feature", worktreeRoot)

	mainFacts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: repositoryRoot,
	})
	if err != nil {
		t.Fatalf("discover main workspace: %v", err)
	}
	worktreeFacts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: filepath.Join(worktreeRoot, "composer.json"),
	})
	if err != nil {
		t.Fatalf("discover linked worktree: %v", err)
	}

	resolvedRepositoryRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	resolvedWorktreeRoot, err := filepath.EvalSymlinks(worktreeRoot)
	if err != nil {
		t.Fatalf("resolve worktree root: %v", err)
	}

	if worktreeFacts.Identity.RepositoryRoot != resolvedRepositoryRoot {
		t.Errorf(
			"expected shared repository root %q, got %q",
			resolvedRepositoryRoot,
			worktreeFacts.Identity.RepositoryRoot,
		)
	}
	if mainFacts.Identity.GitCommonDir != worktreeFacts.Identity.GitCommonDir {
		t.Errorf(
			"expected shared Git common directory, got %q and %q",
			mainFacts.Identity.GitCommonDir,
			worktreeFacts.Identity.GitCommonDir,
		)
	}
	if worktreeFacts.Identity.WorkspaceRoot != resolvedWorktreeRoot {
		t.Errorf(
			"expected worktree workspace root %q, got %q",
			resolvedWorktreeRoot,
			worktreeFacts.Identity.WorkspaceRoot,
		)
	}
	if worktreeFacts.Identity.ComposerRoot != resolvedWorktreeRoot {
		t.Errorf(
			"expected worktree Composer root %q, got %q",
			resolvedWorktreeRoot,
			worktreeFacts.Identity.ComposerRoot,
		)
	}
	if mainFacts.Identity.IdentityKey == worktreeFacts.Identity.IdentityKey {
		t.Errorf("expected distinct workspace identity keys, got %q", mainFacts.Identity.IdentityKey)
	}
	if worktreeFacts.Identity.Branch != "feature" {
		t.Errorf("expected worktree branch feature, got %q", worktreeFacts.Identity.Branch)
	}
}

func TestDiscoverReportsEveryAmbiguousComposerRoot(t *testing.T) {
	repositoryRoot := t.TempDir()
	runGit(t, repositoryRoot, "init", "-b", "main")

	firstRoot := filepath.Join(repositoryRoot, "packages", "api")
	secondRoot := filepath.Join(repositoryRoot, "packages", "worker")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatalf("create first project root: %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("create second project root: %v", err)
	}
	writeFile(t, filepath.Join(firstRoot, "composer.json"), "{}\n")
	writeFile(t, filepath.Join(secondRoot, "composer.json"), "{}\n")

	resolvedFirstRoot, err := filepath.EvalSymlinks(firstRoot)
	if err != nil {
		t.Fatalf("resolve first project root: %v", err)
	}
	resolvedSecondRoot, err := filepath.EvalSymlinks(secondRoot)
	if err != nil {
		t.Fatalf("resolve second project root: %v", err)
	}

	_, err = discovery.Discover(context.Background(), discovery.Request{
		StartPath: repositoryRoot,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if commandError.Code != model.ErrorDiscoveryAmbiguousRoots {
		t.Errorf("expected ambiguity code, got %q", commandError.Code)
	}
	if model.ExitCode(commandError) != 3 {
		t.Errorf("expected discovery exit 3, got %d", model.ExitCode(commandError))
	}

	expectedCandidates := []string{resolvedFirstRoot, resolvedSecondRoot}
	if len(commandError.Details) != len(expectedCandidates) {
		t.Fatalf("expected %d candidate details, got %#v", len(expectedCandidates), commandError.Details)
	}
	for index, expected := range expectedCandidates {
		detail := commandError.Details[index]
		if detail.Name != "candidate" || detail.Value != expected {
			t.Errorf("expected candidate detail %q, got %#v", expected, detail)
		}
	}
}

func TestDiscoverRejectsMalformedComposerMetadataWithSource(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{\n")

	resolvedProjectRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	expectedSource := filepath.Join(resolvedProjectRoot, "composer.json")

	_, err = discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if commandError.Code != model.ErrorDiscovery {
		t.Errorf("expected discovery error code, got %q", commandError.Code)
	}
	if !strings.Contains(commandError.Message, "valid JSON object") {
		t.Errorf("expected invalid JSON message, got %q", commandError.Message)
	}
	if len(commandError.Sources) != 1 {
		t.Fatalf("expected one source reference, got %#v", commandError.Sources)
	}
	if commandError.Sources[0].Path != expectedSource {
		t.Errorf("expected source %q, got %q", expectedSource, commandError.Sources[0].Path)
	}
	if commandError.Sources[0].Kind != "composer_manifest" {
		t.Errorf("expected composer manifest source kind, got %q", commandError.Sources[0].Kind)
	}
}

func TestDiscoverRejectsInvalidUTF8ComposerMetadata(t *testing.T) {
	projectRoot := t.TempDir()
	content := append([]byte(`{"name":"`), 0xff)
	content = append(content, []byte(`"}`)...)
	if err := os.WriteFile(
		filepath.Join(projectRoot, "composer.json"),
		content,
		0o644,
	); err != nil {
		t.Fatalf("write invalid UTF 8 fixture: %v", err)
	}

	_, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if !strings.Contains(commandError.Message, "valid JSON object") {
		t.Errorf("expected invalid JSON message, got %q", commandError.Message)
	}
}

func TestDiscoverRejectsDuplicateComposerMetadataKeys(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(
		t,
		filepath.Join(projectRoot, "composer.json"),
		"{\"config\":{\"platform\":\"first\",\"platform\":\"second\"}}\n",
	)

	_, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if !strings.Contains(commandError.Message, "duplicate object key") {
		t.Errorf("expected duplicate key message, got %q", commandError.Message)
	}
}

func TestDiscoverRejectsUnexpectedComposerMetadataType(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "[]\n")

	_, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if !strings.Contains(commandError.Message, "JSON object") {
		t.Errorf("expected object type message, got %q", commandError.Message)
	}
}

func TestDiscoverRejectsComposerMetadataOverSizeLimit(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{\"name\":\"acme/example\"}\n")

	_, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath:       projectRoot,
		MaxMetadataSize: 16,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if !strings.Contains(commandError.Message, "16 byte limit") {
		t.Errorf("expected size limit message, got %q", commandError.Message)
	}
}

func TestDiscoverRejectsComposerMetadataOverNestingLimit(t *testing.T) {
	projectRoot := t.TempDir()
	content := `{"nested":` +
		strings.Repeat("[", 129) +
		"0" +
		strings.Repeat("]", 129) +
		"}\n"
	writeFile(t, filepath.Join(projectRoot, "composer.json"), content)

	_, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if !strings.Contains(commandError.Message, "nesting limit") {
		t.Errorf("expected nesting limit message, got %q", commandError.Message)
	}
}

func TestDiscoverRejectsComposerSymlinkOutsideProjectBoundary(t *testing.T) {
	projectRoot := t.TempDir()
	outsideRoot := t.TempDir()
	outsideComposer := filepath.Join(outsideRoot, "composer.json")
	writeFile(t, outsideComposer, "{}\n")

	composerPath := filepath.Join(projectRoot, "composer.json")
	if err := os.Symlink(outsideComposer, composerPath); err != nil {
		t.Fatalf("create Composer metadata symlink: %v", err)
	}

	_, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if !strings.Contains(commandError.Message, "outside the project boundary") {
		t.Errorf("expected boundary message, got %q", commandError.Message)
	}
}

func TestDiscoverRejectsComposerLockSymlinkOutsideProjectBoundary(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")

	outsideRoot := t.TempDir()
	outsideLock := filepath.Join(outsideRoot, "composer.lock")
	writeFile(
		t,
		outsideLock,
		`{"content-hash":"99914b932bd37a50b983c5e7c90ae93b","packages":[]}`,
	)
	lockPath := filepath.Join(projectRoot, "composer.lock")
	if err := os.Symlink(outsideLock, lockPath); err != nil {
		t.Fatalf("create Composer lock symlink: %v", err)
	}

	_, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if !strings.Contains(commandError.Message, "outside the project boundary") {
		t.Errorf("expected boundary message, got %q", commandError.Message)
	}
	resolvedProjectRoot, resolveErr := filepath.EvalSymlinks(projectRoot)
	if resolveErr != nil {
		t.Fatalf("resolve project root: %v", resolveErr)
	}
	expectedLockPath := filepath.Join(resolvedProjectRoot, "composer.lock")
	if len(commandError.Sources) != 1 ||
		commandError.Sources[0].Kind != "composer_lock" ||
		commandError.Sources[0].Path != expectedLockPath {
		t.Errorf("expected Composer lock source, got %#v", commandError.Sources)
	}
}

func TestDiscoverFingerprintsComposerInput(t *testing.T) {
	projectRoot := t.TempDir()
	content := "{\"name\":\"acme/example\"}\n"
	writeFile(t, filepath.Join(projectRoot, "composer.json"), content)

	resolvedProjectRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	sum := sha256.Sum256([]byte(content))

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	if len(facts.InputFingerprints) != 1 {
		t.Fatalf("expected one input fingerprint, got %#v", facts.InputFingerprints)
	}
	fingerprint := facts.InputFingerprints[0]
	if fingerprint.Path != filepath.Join(resolvedProjectRoot, "composer.json") {
		t.Errorf("unexpected fingerprint path %q", fingerprint.Path)
	}
	if fingerprint.Kind != "composer_manifest" {
		t.Errorf("unexpected fingerprint kind %q", fingerprint.Kind)
	}
	if fingerprint.SHA256 != hex.EncodeToString(sum[:]) {
		t.Errorf("unexpected fingerprint digest %q", fingerprint.SHA256)
	}
	if fingerprint.Size != int64(len(content)) {
		t.Errorf("expected fingerprint size %d, got %d", len(content), fingerprint.Size)
	}
}

func TestDiscoverRetainsSuppliedPathWhileResolvingIdentity(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")
	descendant := filepath.Join(projectRoot, "src")
	if err := os.MkdirAll(descendant, 0o755); err != nil {
		t.Fatalf("create project descendant: %v", err)
	}

	aliasRoot := filepath.Join(t.TempDir(), "project-alias")
	if err := os.Symlink(projectRoot, aliasRoot); err != nil {
		t.Fatalf("create project symlink: %v", err)
	}
	suppliedPath := filepath.Join(aliasRoot, "src")
	absoluteSuppliedPath, err := filepath.Abs(suppliedPath)
	if err != nil {
		t.Fatalf("make supplied path absolute: %v", err)
	}
	resolvedDescendant, err := filepath.EvalSymlinks(descendant)
	if err != nil {
		t.Fatalf("resolve project descendant: %v", err)
	}
	resolvedProjectRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: suppliedPath,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	if facts.StartingPath.Supplied != suppliedPath {
		t.Errorf("expected supplied path %q, got %q", suppliedPath, facts.StartingPath.Supplied)
	}
	if facts.StartingPath.Absolute != filepath.Clean(absoluteSuppliedPath) {
		t.Errorf(
			"expected absolute supplied path %q, got %q",
			filepath.Clean(absoluteSuppliedPath),
			facts.StartingPath.Absolute,
		)
	}
	if facts.StartingPath.Resolved != resolvedDescendant {
		t.Errorf("expected resolved start path %q, got %q", resolvedDescendant, facts.StartingPath.Resolved)
	}
	if facts.Identity.ComposerRoot != resolvedProjectRoot {
		t.Errorf("expected resolved Composer root %q, got %q", resolvedProjectRoot, facts.Identity.ComposerRoot)
	}
}

func TestDiscoverDoesNotExecuteProjectCode(t *testing.T) {
	projectRoot := t.TempDir()
	markerPath := filepath.Join(projectRoot, "executed")
	composerContent := `{
		"scripts": {
			"post-install-cmd": [
				"php bootstrap.php"
			]
		}
	}
`
	writeFile(t, filepath.Join(projectRoot, "composer.json"), composerContent)
	writeFile(
		t,
		filepath.Join(projectRoot, "bootstrap.php"),
		"<?php file_put_contents("+quotedPHPString(markerPath)+", 'bootstrap');\n",
	)
	if err := os.MkdirAll(filepath.Join(projectRoot, "vendor"), 0o755); err != nil {
		t.Fatalf("create vendor directory: %v", err)
	}
	writeFile(
		t,
		filepath.Join(projectRoot, "vendor", "autoload.php"),
		"<?php file_put_contents("+quotedPHPString(markerPath)+", 'autoload');\n",
	)
	writeFile(
		t,
		filepath.Join(projectRoot, "artisan"),
		"<?php file_put_contents("+quotedPHPString(markerPath)+", 'artisan');\n",
	)

	if _, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	}); err != nil {
		t.Fatalf("discover inert project: %v", err)
	}

	if _, err := os.Stat(markerPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected project code to remain inert, stat error: %v", err)
	}
}

func TestDiscoverStopsWhenContextIsCanceled(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := discovery.Discover(ctx, discovery.Request{
		StartPath: projectRoot,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled discovery, got %v", err)
	}
}

func TestDiscoverParsesComposerAndLockFacts(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(
		t,
		filepath.Join(projectRoot, "composer.json"),
		string(readComposerFixture(t, "locked-platform", "composer.json")),
	)
	writeFile(
		t,
		filepath.Join(projectRoot, "composer.lock"),
		string(readComposerFixture(t, "locked-platform", "composer.lock")),
	)

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	if facts.Composer.Manifest.Name != "acme/locked-platform" {
		t.Errorf("unexpected Composer package name %q", facts.Composer.Manifest.Name)
	}
	if facts.Composer.Lock.Status != model.ComposerLockFresh {
		t.Errorf("expected fresh Composer lock, got %q", facts.Composer.Lock.Status)
	}
	if len(facts.Composer.PlatformRequirements) != 6 {
		t.Errorf(
			"expected root and locked platform requirements, got %#v",
			facts.Composer.PlatformRequirements,
		)
	}
	if len(facts.Diagnostics) != 0 {
		t.Errorf("expected no Composer diagnostics, got %#v", facts.Diagnostics)
	}
	if len(facts.InputFingerprints) != 2 {
		t.Fatalf("expected manifest and lock fingerprints, got %#v", facts.InputFingerprints)
	}
	if facts.InputFingerprints[0].Kind != "composer_manifest" {
		t.Errorf("unexpected first fingerprint kind %q", facts.InputFingerprints[0].Kind)
	}
	if facts.InputFingerprints[1].Kind != "composer_lock" {
		t.Errorf("unexpected second fingerprint kind %q", facts.InputFingerprints[1].Kind)
	}
}

func TestDiscoverAddsFrameworkFactsWithoutBootingApplication(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join(
		"..",
		"..",
		"testdata",
		"fixtures",
		"frameworks",
		"laravel-application",
	))
	if err != nil {
		t.Fatalf("resolve framework fixture: %v", err)
	}

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	var laravel *model.FrameworkFact
	for index := range facts.Frameworks {
		if facts.Frameworks[index].Kind == model.FrameworkLaravelApplication {
			laravel = &facts.Frameworks[index]
			break
		}
	}
	if laravel == nil {
		t.Fatalf("expected Laravel application fact, got %#v", facts.Frameworks)
	}
	if !laravel.Primary {
		t.Fatal("expected Laravel application to be primary")
	}
}

func TestDiscoverAddsVersionAndProviderMarkerFactsWithFingerprints(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")
	writeFile(t, filepath.Join(projectRoot, ".php-version"), "8.4\n")
	if err := os.MkdirAll(filepath.Join(projectRoot, ".ddev"), 0o755); err != nil {
		t.Fatalf("create DDEV directory: %v", err)
	}
	writeFile(
		t,
		filepath.Join(projectRoot, ".ddev", "config.yaml"),
		"name: example\n",
	)
	writeFile(t, filepath.Join(projectRoot, "herd.yml"), "php: 8.4\n")

	resolvedProjectRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	if len(facts.VersionFiles) != 1 {
		t.Fatalf("expected one version file fact, got %#v", facts.VersionFiles)
	}
	version := facts.VersionFiles[0]
	if version.Runtime != "php" || version.Version != "8.4" {
		t.Fatalf("unexpected PHP version fact %#v", version)
	}
	expectedVersionPath := filepath.Join(resolvedProjectRoot, ".php-version")
	if version.Source.Path != expectedVersionPath {
		t.Fatalf("expected version source %q, got %#v", expectedVersionPath, version.Source)
	}

	if len(facts.ProviderMarkers) != 2 {
		t.Fatalf("expected two provider markers, got %#v", facts.ProviderMarkers)
	}
	if facts.ProviderMarkers[0].Provider != "ddev" ||
		facts.ProviderMarkers[1].Provider != "herd" {
		t.Fatalf("unexpected provider markers %#v", facts.ProviderMarkers)
	}
	if !hasInputFingerprint(
		facts.InputFingerprints,
		expectedVersionPath,
		"php_version",
	) {
		t.Fatalf("expected PHP version fingerprint, got %#v", facts.InputFingerprints)
	}
	if !hasInputFingerprint(
		facts.InputFingerprints,
		filepath.Join(resolvedProjectRoot, ".ddev", "config.yaml"),
		"provider_config",
	) {
		t.Fatalf("expected DDEV fingerprint, got %#v", facts.InputFingerprints)
	}
}

func TestDiscoverRejectsVersionFileSymlinkOutsideProjectBoundary(t *testing.T) {
	projectRoot := t.TempDir()
	outsideRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")
	outsideVersion := filepath.Join(outsideRoot, ".php-version")
	writeFile(t, outsideVersion, "8.4\n")
	if err := os.Symlink(
		outsideVersion,
		filepath.Join(projectRoot, ".php-version"),
	); err != nil {
		t.Fatalf("create version file symlink: %v", err)
	}

	_, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if !strings.Contains(commandError.Message, "outside the project boundary") {
		t.Fatalf("expected boundary error, got %#v", commandError)
	}
	if len(commandError.Sources) != 1 ||
		commandError.Sources[0].Kind != "php_version" {
		t.Fatalf("expected PHP version source, got %#v", commandError.Sources)
	}
}

func TestDiscoverRejectsInvalidUTF8PHPVersionFile(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")
	if err := os.WriteFile(
		filepath.Join(projectRoot, ".php-version"),
		[]byte{0xff, '\n'},
		0o644,
	); err != nil {
		t.Fatalf("write invalid PHP version file: %v", err)
	}

	_, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	if len(commandError.Sources) != 1 ||
		commandError.Sources[0].Kind != "php_version" {
		t.Fatalf("expected PHP version source, got %#v", commandError.Sources)
	}
}

func hasInputFingerprint(
	fingerprints []model.InputFingerprint,
	path string,
	kind string,
) bool {
	for _, fingerprint := range fingerprints {
		if fingerprint.Path == path && fingerprint.Kind == kind {
			return true
		}
	}

	return false
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readComposerFixture(t *testing.T, parts ...string) []byte {
	t.Helper()

	pathParts := append(
		[]string{"..", "..", "testdata", "fixtures", "composer"},
		parts...,
	)
	content, err := os.ReadFile(filepath.Join(pathParts...))
	if err != nil {
		t.Fatalf("read Composer fixture: %v", err)
	}

	return content
}

func quotedPHPString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}

func runGit(t *testing.T, directory string, arguments ...string) string {
	t.Helper()

	command := exec.Command("git", arguments...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\noutput:\n%s", strings.Join(arguments, " "), err, output)
	}

	return strings.TrimSpace(string(output))
}
