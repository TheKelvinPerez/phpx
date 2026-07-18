package state_test

import (
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/state"
)

func TestTrustRecordRequiresExactCurrentFingerprints(t *testing.T) {
	t.Parallel()

	record := state.NewTrustRecord("sha256:project")
	approved := []model.TrustRequirement{
		{
			Class:       model.TrustComposerPlugins,
			Fingerprint: "sha256:original",
		},
		{
			Class:       model.TrustComposerScripts,
			Fingerprint: "sha256:original",
		},
	}
	record.Approve(approved)

	if missing := record.Missing(approved); len(missing) != 0 {
		t.Fatalf("approved requirements reported missing: %#v", missing)
	}

	changed := append([]model.TrustRequirement(nil), approved...)
	changed[1].Fingerprint = "sha256:changed"
	missing := record.Missing(changed)
	if len(missing) != 1 {
		t.Fatalf("expected one invalidated approval, got %#v", missing)
	}
	if missing[0].Class != model.TrustComposerScripts ||
		missing[0].Fingerprint != "sha256:changed" {
		t.Fatalf("unexpected invalidated trust requirement %#v", missing[0])
	}
}
