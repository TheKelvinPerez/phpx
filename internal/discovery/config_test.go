package discovery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/elefantephp/elefante/internal/discovery"
	"github.com/elefantephp/elefante/internal/model"
)

func TestDiscoverUsesExplicitConfigurationAndFingerprintsIt(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")
	writeFile(
		t,
		filepath.Join(projectRoot, "elefante.toml"),
		"schema_version = 1\n[composer]\nconstraint = \"^1\"\n",
	)
	explicitPath := filepath.Join(projectRoot, "custom.toml")
	writeFile(
		t,
		explicitPath,
		"schema_version = 1\n[composer]\nconstraint = \"^2\"\n",
	)
	resolvedExplicitPath, err := filepath.EvalSymlinks(explicitPath)
	if err != nil {
		t.Fatalf("resolve explicit config: %v", err)
	}

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath:  projectRoot,
		ConfigPath: explicitPath,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	if facts.Configuration.Path != resolvedExplicitPath {
		t.Fatalf(
			"expected explicit config %q, got %q",
			resolvedExplicitPath,
			facts.Configuration.Path,
		)
	}
	if facts.Configuration.Composer.Constraint != "^2" {
		t.Fatalf("expected explicit Composer policy, got %#v", facts.Configuration.Composer)
	}
	if !hasFingerprint(facts.InputFingerprints, resolvedExplicitPath, "elefante_config") {
		t.Fatalf("expected config fingerprint, got %#v", facts.InputFingerprints)
	}
}

func TestDiscoverUsesRepositoryConfigurationToSelectComposerRoot(t *testing.T) {
	repositoryRoot := t.TempDir()
	runGit(t, repositoryRoot, "init", "-b", "main")
	selectedRoot := filepath.Join(repositoryRoot, "apps", "web")
	otherRoot := filepath.Join(repositoryRoot, "packages", "library")
	if err := mkdirAll(selectedRoot, otherRoot); err != nil {
		t.Fatalf("create Composer roots: %v", err)
	}
	writeFile(t, filepath.Join(selectedRoot, "composer.json"), "{}\n")
	writeFile(t, filepath.Join(otherRoot, "composer.json"), "{}\n")
	configPath := filepath.Join(repositoryRoot, "elefante.toml")
	writeFile(
		t,
		configPath,
		"schema_version = 1\n[project]\ncomposer_root = \"apps/web\"\n",
	)
	resolvedSelectedRoot, err := filepath.EvalSymlinks(selectedRoot)
	if err != nil {
		t.Fatalf("resolve selected Composer root: %v", err)
	}
	resolvedConfigPath, err := filepath.EvalSymlinks(configPath)
	if err != nil {
		t.Fatalf("resolve repository config: %v", err)
	}

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: repositoryRoot,
	})
	if err != nil {
		t.Fatalf("discover configured monorepo project: %v", err)
	}

	if facts.Identity.ComposerRoot != resolvedSelectedRoot {
		t.Fatalf(
			"expected configured Composer root %q, got %q",
			resolvedSelectedRoot,
			facts.Identity.ComposerRoot,
		)
	}
	if facts.Configuration.Path != resolvedConfigPath {
		t.Fatalf(
			"expected repository config %q, got %q",
			resolvedConfigPath,
			facts.Configuration.Path,
		)
	}
}

func TestDiscoverRepositoryConfigurationOverridesRootComposerProject(t *testing.T) {
	repositoryRoot := t.TempDir()
	runGit(t, repositoryRoot, "init", "-b", "main")
	selectedRoot := filepath.Join(repositoryRoot, "apps", "web")
	if err := mkdirAll(selectedRoot); err != nil {
		t.Fatalf("create nested Composer root: %v", err)
	}
	writeFile(t, filepath.Join(repositoryRoot, "composer.json"), "{}\n")
	writeFile(t, filepath.Join(selectedRoot, "composer.json"), "{}\n")
	writeFile(
		t,
		filepath.Join(repositoryRoot, "elefante.toml"),
		"schema_version = 1\n[project]\ncomposer_root = \"apps/web\"\n",
	)
	resolvedSelectedRoot, err := filepath.EvalSymlinks(selectedRoot)
	if err != nil {
		t.Fatalf("resolve selected Composer root: %v", err)
	}

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: repositoryRoot,
	})
	if err != nil {
		t.Fatalf("discover configured monorepo project: %v", err)
	}

	if facts.Identity.ComposerRoot != resolvedSelectedRoot {
		t.Fatalf(
			"expected configured Composer root %q, got %q",
			resolvedSelectedRoot,
			facts.Identity.ComposerRoot,
		)
	}
}

func TestDiscoverUsesProjectConfigurationWhenRepositorySelectsIt(t *testing.T) {
	repositoryRoot := t.TempDir()
	runGit(t, repositoryRoot, "init", "-b", "main")
	projectRoot := filepath.Join(repositoryRoot, "apps", "web")
	if err := mkdirAll(projectRoot); err != nil {
		t.Fatalf("create project root: %v", err)
	}
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")
	writeFile(
		t,
		filepath.Join(repositoryRoot, "elefante.toml"),
		"schema_version = 1\n[project]\ncomposer_root = \"apps/web\"\n",
	)
	projectConfig := filepath.Join(projectRoot, "elefante.toml")
	writeFile(
		t,
		projectConfig,
		"schema_version = 1\n[composer]\nconstraint = \"^2\"\n",
	)
	resolvedProjectConfig, err := filepath.EvalSymlinks(projectConfig)
	if err != nil {
		t.Fatalf("resolve project config: %v", err)
	}

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	if facts.Configuration.Path != resolvedProjectConfig {
		t.Fatalf(
			"expected project config %q, got %q",
			resolvedProjectConfig,
			facts.Configuration.Path,
		)
	}
	if facts.Configuration.Composer.Constraint != "^2" {
		t.Fatalf("expected project Composer policy, got %#v", facts.Configuration.Composer)
	}
}

func TestDiscoverReportsAmbiguousRepositoryAndProjectConfiguration(t *testing.T) {
	repositoryRoot := t.TempDir()
	runGit(t, repositoryRoot, "init", "-b", "main")
	projectRoot := filepath.Join(repositoryRoot, "apps", "web")
	if err := mkdirAll(projectRoot); err != nil {
		t.Fatalf("create project root: %v", err)
	}
	writeFile(t, filepath.Join(projectRoot, "composer.json"), "{}\n")
	writeFile(
		t,
		filepath.Join(repositoryRoot, "elefante.toml"),
		"schema_version = 1\n",
	)
	writeFile(
		t,
		filepath.Join(projectRoot, "elefante.toml"),
		"schema_version = 1\n",
	)

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: projectRoot,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	if facts.Configuration.Path != "" {
		t.Fatalf("expected no arbitrarily selected config, got %#v", facts.Configuration)
	}
	if !hasDiagnostic(facts.Diagnostics, "ELEFANTE_CONFIG_AMBIGUOUS") {
		t.Fatalf("expected config ambiguity diagnostic, got %#v", facts.Diagnostics)
	}
}

func TestDiscoverReportsConfigurationComposerRootMismatch(t *testing.T) {
	repositoryRoot := t.TempDir()
	runGit(t, repositoryRoot, "init", "-b", "main")
	writeFile(t, filepath.Join(repositoryRoot, "composer.json"), "{}\n")
	if err := mkdirAll(filepath.Join(repositoryRoot, "apps", "missing")); err != nil {
		t.Fatalf("create configured directory: %v", err)
	}
	writeFile(
		t,
		filepath.Join(repositoryRoot, "elefante.toml"),
		"schema_version = 1\n[project]\ncomposer_root = \"apps/missing\"\n",
	)

	facts, err := discovery.Discover(context.Background(), discovery.Request{
		StartPath: repositoryRoot,
	})
	if err != nil {
		t.Fatalf("discover project: %v", err)
	}

	if !hasDiagnostic(facts.Diagnostics, "ELEFANTE_CONFIG_PROJECT_MISMATCH") {
		t.Fatalf("expected project mismatch diagnostic, got %#v", facts.Diagnostics)
	}
}

func hasFingerprint(
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

func hasDiagnostic(
	diagnostics []model.Diagnostic,
	code string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}

	return false
}

func mkdirAll(paths ...string) error {
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}

	return nil
}
