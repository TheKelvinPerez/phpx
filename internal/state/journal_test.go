package state_test

import (
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/state"
)

func TestActionJournalTracksCompletedFailedAndPendingActions(t *testing.T) {
	t.Parallel()

	journal := state.NewActionJournal(
		"sha256:project",
		"sha256:plan",
		[]model.PlanAction{
			{ID: "prepare-provider", Kind: model.ActionPrepareProvider},
			{ID: "install-dependencies", Kind: model.ActionInstallDependencies},
			{ID: "record-state", Kind: model.ActionRecordState},
		},
	)
	if err := journal.Complete("prepare-provider"); err != nil {
		t.Fatalf("complete action: %v", err)
	}
	if err := journal.Fail("install-dependencies", "Composer exited with status 1."); err != nil {
		t.Fatalf("fail action: %v", err)
	}

	if journal.SchemaVersion != state.ActionJournalSchemaVersion {
		t.Fatalf("unexpected schema %q", journal.SchemaVersion)
	}
	if journal.Status != state.JournalFailed {
		t.Fatalf("expected failed journal, got %q", journal.Status)
	}
	assertJournalAction(t, journal, "prepare-provider", state.ActionCompleted)
	assertJournalAction(t, journal, "install-dependencies", state.ActionFailed)
	assertJournalAction(t, journal, "record-state", state.ActionPending)
}

func TestActionJournalBecomesSuccessfulOnlyAfterEveryActionCompletes(t *testing.T) {
	t.Parallel()

	journal := state.NewActionJournal(
		"sha256:project",
		"sha256:plan",
		[]model.PlanAction{
			{ID: "prepare", Kind: model.ActionPrepareProvider},
			{ID: "record", Kind: model.ActionRecordState},
		},
	)
	if err := journal.Complete("prepare"); err != nil {
		t.Fatalf("complete first action: %v", err)
	}
	if err := journal.Succeed(); err == nil {
		t.Fatal("expected pending action to block successful journal")
	}
	if err := journal.Complete("record"); err != nil {
		t.Fatalf("complete final action: %v", err)
	}
	if err := journal.Succeed(); err != nil {
		t.Fatalf("complete journal: %v", err)
	}
	if journal.Status != state.JournalSucceeded {
		t.Fatalf("expected successful journal, got %q", journal.Status)
	}
}

func assertJournalAction(
	t *testing.T,
	journal state.ActionJournal,
	id string,
	expected state.ActionStatus,
) {
	t.Helper()

	for _, action := range journal.Actions {
		if action.ID == id {
			if action.Status != expected {
				t.Fatalf("expected %s status %q, got %q", id, expected, action.Status)
			}

			return
		}
	}

	t.Fatalf("missing action %q in %#v", id, journal.Actions)
}
