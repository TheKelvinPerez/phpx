package security

import (
	"crypto/sha256"
	"crypto/subtle"

	"github.com/elefantephp/elefante/internal/model"
)

type ApprovalOptions struct {
	Yes          bool
	ApprovedPlan string
	Confirmed    bool
}

func AuthorizePlan(plan model.Plan, options ApprovalOptions) error {
	if options.Yes && options.ApprovedPlan != "" {
		return model.NewError(
			model.ErrorUsage,
			"The --yes and --approve-plan flags cannot be used together.",
		).WithHint("Choose one approval method and retry.")
	}

	if options.ApprovedPlan != "" &&
		!constantTimeEqual(options.ApprovedPlan, plan.Digest) {
		commandError := model.NewError(
			model.ErrorPlanMismatch,
			"The reviewed plan no longer matches the current project.",
		).WithHint("Review the new plan and approve its exact digest.")
		commandError.Details = []model.ErrorDetail{
			{Name: "expected_digest", Value: options.ApprovedPlan},
			{Name: "actual_digest", Value: plan.Digest},
			{Name: "changed_categories", Value: "plan_content"},
		}

		return commandError
	}

	if !PlanRequiresApproval(plan) ||
		options.Yes ||
		options.ApprovedPlan != "" ||
		options.Confirmed {
		return nil
	}

	commandError := model.NewError(
		model.ErrorApprovalRequired,
		"The synchronization plan requires explicit approval.",
	).WithHint(
		"Review elefante plan, then use --approve-plan with its exact digest or use --yes.",
	)
	commandError.Details = []model.ErrorDetail{
		{Name: "plan_digest", Value: plan.Digest},
	}

	return commandError
}

func PlanRequiresApproval(plan model.Plan) bool {
	for _, action := range plan.Actions {
		if action.Effect != model.EffectRead {
			return true
		}
	}

	return len(plan.Trust) > 0
}

func constantTimeEqual(left string, right string) bool {
	leftDigest := sha256.Sum256([]byte(left))
	rightDigest := sha256.Sum256([]byte(right))

	return subtle.ConstantTimeCompare(leftDigest[:], rightDigest[:]) == 1
}
