package security_test

import (
	"errors"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/security"
)

func TestAuthorizePlanRejectsMismatchedExactDigest(t *testing.T) {
	t.Parallel()

	err := security.AuthorizePlan(
		mutatingPlan("sha256:actual"),
		security.ApprovalOptions{
			ApprovedPlan: "sha256:reviewed",
		},
	)

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected a public command error, got %v", err)
	}
	if commandError.Code != model.ErrorPlanMismatch {
		t.Fatalf("expected %s, got %#v", model.ErrorPlanMismatch, commandError)
	}
	if model.ExitCode(err) != 7 {
		t.Fatalf("expected exit 7, got %d", model.ExitCode(err))
	}
	assertErrorDetail(t, commandError, "expected_digest", "sha256:reviewed")
	assertErrorDetail(t, commandError, "actual_digest", "sha256:actual")
}

func TestAuthorizePlanEnforcesExplicitApprovalMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plan     model.Plan
		options  security.ApprovalOptions
		wantCode model.ErrorCode
	}{
		{
			name:     "mutation without approval",
			plan:     mutatingPlan("sha256:actual"),
			wantCode: model.ErrorApprovalRequired,
		},
		{
			name:    "yes approves freshly computed plan",
			plan:    mutatingPlan("sha256:actual"),
			options: security.ApprovalOptions{Yes: true},
		},
		{
			name: "exact digest approves only current plan",
			plan: mutatingPlan("sha256:actual"),
			options: security.ApprovalOptions{
				ApprovedPlan: "sha256:actual",
			},
		},
		{
			name:    "interactive confirmation approves current plan",
			plan:    mutatingPlan("sha256:actual"),
			options: security.ApprovalOptions{Confirmed: true},
		},
		{
			name: "read only plan needs no approval",
			plan: model.Plan{
				Digest: "sha256:read",
				Actions: []model.PlanAction{
					{ID: "inspect", Effect: model.EffectRead},
				},
			},
		},
		{
			name: "trust requirement needs approval",
			plan: model.Plan{
				Digest: "sha256:trust",
				Trust: []model.TrustRequirement{
					{
						Class:       model.TrustComposerScripts,
						Fingerprint: "sha256:scripts",
					},
				},
			},
			wantCode: model.ErrorApprovalRequired,
		},
		{
			name: "approval flags are mutually exclusive",
			plan: mutatingPlan("sha256:actual"),
			options: security.ApprovalOptions{
				Yes:          true,
				ApprovedPlan: "sha256:actual",
			},
			wantCode: model.ErrorUsage,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := security.AuthorizePlan(test.plan, test.options)
			if test.wantCode == "" {
				if err != nil {
					t.Fatalf("authorize plan: %v", err)
				}

				return
			}
			var commandError *model.Error
			if !errors.As(err, &commandError) ||
				commandError.Code != test.wantCode {
				t.Fatalf("expected %s, got %v", test.wantCode, err)
			}
		})
	}
}

func mutatingPlan(digest string) model.Plan {
	return model.Plan{
		SchemaVersion: model.PlanSchemaVersion,
		Operation:     model.OperationSync,
		Actions: []model.PlanAction{
			{
				ID:     "record-state",
				Kind:   model.ActionRecordState,
				Effect: model.EffectLocalStateMutation,
			},
		},
		Digest: digest,
	}
}

func assertErrorDetail(
	t *testing.T,
	commandError *model.Error,
	name string,
	expected string,
) {
	t.Helper()

	for _, detail := range commandError.Details {
		if detail.Name == name {
			if detail.Value != expected {
				t.Fatalf(
					"expected %s detail %q, got %q",
					name,
					expected,
					detail.Value,
				)
			}

			return
		}
	}

	t.Fatalf("missing %s detail in %#v", name, commandError.Details)
}
