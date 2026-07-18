package providers

import (
	"context"

	"github.com/elefantephp/elefante/internal/model"
)

type InspectRequest struct {
	Project model.ProjectIdentity `json:"project"`
	Offline bool                  `json:"offline"`
}

type ProviderPlanRequest struct {
	Facts       model.ProjectFacts        `json:"facts"`
	Observation model.ProviderObservation `json:"observation"`
	Policy      model.PlanPolicy          `json:"policy"`
}

type ProviderPlan struct {
	Actions     []model.PlanAction `json:"actions"`
	Diagnostics []model.Diagnostic `json:"diagnostics,omitempty"`
}

type ProviderAction struct {
	Action model.PlanAction `json:"action"`
}

type ActionRuntime struct {
	Environment []string `json:"environment,omitempty"`
}

type ActionResult struct {
	Outputs      []model.ActionOutput `json:"outputs"`
	Diagnostics  []model.Diagnostic   `json:"diagnostics,omitempty"`
	Compensation *ActionCompensation  `json:"compensation,omitempty"`
}

type ActionCompensation struct {
	Safe   bool             `json:"safe"`
	Action model.PlanAction `json:"action"`
}

type ExecutionRequest struct {
	Executable       string   `json:"executable"`
	Arguments        []string `json:"arguments"`
	WorkingDirectory string   `json:"working_directory"`
	Environment      []string `json:"environment,omitempty"`
}

type InputMode string

const (
	InputInherit InputMode = "inherit"
	InputClosed  InputMode = "closed"
)

type OutputMode string

const (
	OutputStream  OutputMode = "stream"
	OutputCapture OutputMode = "capture"
)

type ExecutionSpec struct {
	Executable       string     `json:"executable"`
	Arguments        []string   `json:"arguments"`
	WorkingDirectory string     `json:"working_directory"`
	Environment      []string   `json:"environment,omitempty"`
	InputMode        InputMode  `json:"input_mode"`
	OutputMode       OutputMode `json:"output_mode"`
}

type Provider interface {
	Name() string
	Inspect(context.Context, InspectRequest) (model.ProviderObservation, error)
	Plan(context.Context, ProviderPlanRequest) (ProviderPlan, error)
	Apply(
		context.Context,
		ProviderAction,
		ActionRuntime,
	) (ActionResult, error)
	ExecutionSpec(context.Context, ExecutionRequest) (ExecutionSpec, error)
}
