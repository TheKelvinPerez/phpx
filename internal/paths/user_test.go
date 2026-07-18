package paths_test

import (
	"path/filepath"
	"testing"

	"github.com/elefantephp/elefante/internal/paths"
)

func TestResolveUserPathsUsesMacOSConventions(t *testing.T) {
	t.Parallel()

	home := filepath.Join(string(filepath.Separator), "Users", "kelvin")
	resolved, err := paths.ResolveUserPaths(paths.UserPathOptions{
		GOOS: "darwin",
		Home: home,
	})
	if err != nil {
		t.Fatalf("resolve user paths: %v", err)
	}

	assertPath(t, resolved.ConfigRoot, filepath.Join(
		home,
		"Library",
		"Application Support",
		"Elefante",
	))
	assertPath(t, resolved.CacheRoot, filepath.Join(
		home,
		"Library",
		"Caches",
		"Elefante",
	))
	assertPath(t, resolved.LogRoot, filepath.Join(
		home,
		"Library",
		"Logs",
		"Elefante",
	))

	project, err := resolved.Project("sha256:project")
	if err != nil {
		t.Fatalf("resolve project state paths: %v", err)
	}
	stateRoot := filepath.Join(
		resolved.ConfigRoot,
		"state",
		"projects",
		"sha256_project",
	)
	assertPath(t, project.Environment, filepath.Join(stateRoot, "environment.json"))
	assertPath(t, project.Trust, filepath.Join(stateRoot, "trust.json"))
	assertPath(t, project.Journal, filepath.Join(stateRoot, "journal.json"))
	assertPath(
		t,
		project.Lock,
		filepath.Join(resolved.ConfigRoot, "locks", "sha256_project.lock"),
	)
}

func assertPath(t *testing.T, actual string, expected string) {
	t.Helper()

	if actual != expected {
		t.Fatalf("expected path %q, got %q", expected, actual)
	}
}
