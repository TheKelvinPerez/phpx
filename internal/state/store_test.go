package state_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/paths"
	"github.com/elefantephp/elefante/internal/security"
	"github.com/elefantephp/elefante/internal/state"
)

func TestStorePersistsVersionedRedactedState(t *testing.T) {
	t.Parallel()

	const secret = "state-file-secret"
	userPaths := testUserPaths(t)
	store := state.NewStore(userPaths, security.NewRedactor(secret))
	journal := state.NewActionJournal(
		"sha256:project",
		"sha256:plan",
		[]model.PlanAction{
			{ID: "install", Kind: model.ActionInstallDependencies},
		},
	)
	if err := journal.Fail("install", "Authorization: Bearer "+secret); err != nil {
		t.Fatalf("mark action failed: %v", err)
	}

	if err := store.SaveJournal("sha256:project", journal); err != nil {
		t.Fatalf("save journal: %v", err)
	}
	projectPaths, err := userPaths.Project("sha256:project")
	if err != nil {
		t.Fatalf("resolve project paths: %v", err)
	}
	content, err := os.ReadFile(projectPaths.Journal)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	if strings.Contains(string(content), secret) {
		t.Fatalf("journal leaked a secret: %s", content)
	}

	loaded, exists, err := store.LoadJournal("sha256:project")
	if err != nil {
		t.Fatalf("load journal: %v", err)
	}
	if !exists {
		t.Fatal("expected persisted journal")
	}
	if loaded.SchemaVersion != state.ActionJournalSchemaVersion ||
		loaded.Actions[0].Failure != "Authorization: Bearer [REDACTED]" {
		t.Fatalf("unexpected persisted journal %#v", loaded)
	}
}

func TestStoreRejectsFutureSchemaWithoutDeletingState(t *testing.T) {
	t.Parallel()

	userPaths := testUserPaths(t)
	store := state.NewStore(userPaths, security.NewRedactor())
	projectPaths, err := userPaths.Project("sha256:project")
	if err != nil {
		t.Fatalf("resolve project paths: %v", err)
	}
	if err := state.WriteJSON(projectPaths.Trust, map[string]any{
		"schema_version":   "elefante.trust/v99",
		"project_identity": "sha256:project",
		"approvals":        []any{},
	}); err != nil {
		t.Fatalf("write future trust state: %v", err)
	}

	_, _, err = store.LoadTrust("sha256:project")
	var commandError *model.Error
	if !errors.As(err, &commandError) ||
		commandError.Code != model.ErrorState {
		t.Fatalf("expected state error, got %v", err)
	}
	if _, statErr := os.Stat(projectPaths.Trust); statErr != nil {
		t.Fatalf("future state must remain available for recovery: %v", statErr)
	}
}

func testUserPaths(t *testing.T) paths.UserPaths {
	t.Helper()

	configRoot := filepath.Join(t.TempDir(), "config")

	return paths.UserPaths{
		ConfigRoot: configRoot,
		CacheRoot:  filepath.Join(t.TempDir(), "cache"),
		LogRoot:    filepath.Join(t.TempDir(), "logs"),
		ConfigFile: filepath.Join(configRoot, "config.toml"),
		StateRoot:  filepath.Join(configRoot, "state"),
		LocksRoot:  filepath.Join(configRoot, "locks"),
	}
}
