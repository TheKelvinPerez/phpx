package model_test

import (
	"encoding/json"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
)

func TestPlanPublicNamesRemainStable(t *testing.T) {
	if model.PlanSchemaVersion != "elefante.plan/v1" {
		t.Fatalf("unexpected plan schema %q", model.PlanSchemaVersion)
	}

	actionKinds := []model.ActionKind{
		model.ActionPrepareCache,
		model.ActionPrepareRuntime,
		model.ActionPrepareExtension,
		model.ActionPrepareComposer,
		model.ActionPrepareProvider,
		model.ActionInstallDependencies,
		model.ActionVerifyPlatform,
		model.ActionRecordState,
	}
	expectedActions := []string{
		"prepare_cache",
		"prepare_runtime",
		"prepare_extension",
		"prepare_composer",
		"prepare_provider",
		"install_dependencies",
		"verify_platform",
		"record_state",
	}
	for index, kind := range actionKinds {
		if string(kind) != expectedActions[index] {
			t.Fatalf(
				"expected action kind %q, got %q",
				expectedActions[index],
				kind,
			)
		}
	}

	statuses := []model.ResolutionStatus{
		model.ResolutionSatisfied,
		model.ResolutionActionRequired,
		model.ResolutionBlocked,
		model.ResolutionAmbiguous,
		model.ResolutionLegacy,
		model.ResolutionOptionalMissing,
	}
	expectedStatuses := []string{
		"satisfied",
		"action_required",
		"blocked",
		"ambiguous",
		"legacy",
		"optional_missing",
	}
	for index, status := range statuses {
		if string(status) != expectedStatuses[index] {
			t.Fatalf(
				"expected resolution status %q, got %q",
				expectedStatuses[index],
				status,
			)
		}
	}
}

func TestPlanMarshalsBehaviorPolicyAndDigest(t *testing.T) {
	value := model.Plan{
		SchemaVersion: model.PlanSchemaVersion,
		Operation:     model.OperationSync,
		Actions:       []model.PlanAction{},
		Requirements:  []model.RequirementResolution{},
		Inputs:        []model.InputFingerprint{},
		Policy: model.PlanPolicy{
			Offline: true,
			Frozen:  true,
		},
		Digest: "sha256:digest",
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	expected := `{"schema_version":"elefante.plan/v1","operation":"sync","project":{"composer_root":"","application_root":"","workspace_root":""},"provider":{},"requirements":[],"actions":[],"inputs":[],"policy":{"offline":true,"frozen":true},"digest":"sha256:digest"}`
	if string(encoded) != expected {
		t.Fatalf(
			"expected stable plan JSON\nexpected: %s\ngot:      %s",
			expected,
			encoded,
		)
	}
}
