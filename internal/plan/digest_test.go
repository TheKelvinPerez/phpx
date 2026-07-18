package plan

import (
	"testing"

	"github.com/elefantephp/elefante/internal/model"
)

func TestComputeDigestExcludesAllDisplayWording(t *testing.T) {
	value := model.Plan{
		SchemaVersion: model.PlanSchemaVersion,
		Operation:     model.OperationSync,
		Actions: []model.PlanAction{
			{
				ID:      "01-verify",
				Kind:    model.ActionVerifyPlatform,
				Summary: "First action wording.",
				Effect:  model.EffectRead,
				Network: model.NetworkNone,
				Trust:   model.TrustNone,
				Inputs: []model.ActionInput{
					{Name: "provider", Value: "native"},
				},
			},
		},
		Diagnostics: []model.Diagnostic{
			{
				Code:     "ELEFANTE_FIXTURE",
				Severity: model.SeverityWarning,
				Message:  "First message.",
				Detail:   "First detail.",
				Hint:     "First hint.",
			},
		},
		Inputs: []model.InputFingerprint{},
		Policy: model.PlanPolicy{},
	}
	first, err := computeDigest(value)
	if err != nil {
		t.Fatalf("compute first digest: %v", err)
	}

	value.Actions[0].Summary = "Completely different action wording."
	value.Diagnostics[0].Message = "Completely different message."
	value.Diagnostics[0].Detail = "Completely different detail."
	value.Diagnostics[0].Hint = "Completely different hint."
	second, err := computeDigest(value)
	if err != nil {
		t.Fatalf("compute second digest: %v", err)
	}
	if first != second {
		t.Fatalf("display wording changed digest from %q to %q", first, second)
	}

	value.Actions[0].Inputs[0].Value = "ddev"
	changed, err := computeDigest(value)
	if err != nil {
		t.Fatalf("compute changed digest: %v", err)
	}
	if changed == second {
		t.Fatalf("normalized action input did not change digest %q", changed)
	}
}
