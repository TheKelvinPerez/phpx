package providers_test

import (
	"context"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
	"github.com/elefantephp/elefante/internal/providers/providertest"
)

func TestProviderConformance(t *testing.T) {
	providertest.Run(t, providertest.Suite{
		Provider: &fakeProvider{},
		InspectRequest: providers.InspectRequest{
			Project: model.ProjectIdentity{
				ComposerRoot:  "/workspace",
				WorkspaceRoot: "/workspace",
				IdentityKey:   "sha256:project",
			},
			Offline: true,
		},
		ExecutionRequest: providers.ExecutionRequest{
			Executable:       "php",
			Arguments:        []string{"-v", "--fixture"},
			WorkingDirectory: "/workspace",
			Environment:      []string{"APP_ENV=test"},
		},
	})
}

type fakeProvider struct{}

func (provider *fakeProvider) Name() string {
	return "fake"
}

func (provider *fakeProvider) Inspect(
	context.Context,
	providers.InspectRequest,
) (model.ProviderObservation, error) {
	return model.ProviderObservation{
		Provider:  "fake",
		Available: true,
		Capabilities: []model.Capability{
			model.CapabilityExecuteCommand,
			model.CapabilityInspectRuntime,
		},
		Runtimes: []model.RuntimeObservation{
			{
				Name:    "php",
				Version: "8.5.0",
				Source: model.SourceReference{
					Path: "/fixture/bin/php",
					Kind: "provider_executable",
				},
			},
		},
		Fingerprint: "sha256:fake",
	}, nil
}

func (provider *fakeProvider) Plan(
	context.Context,
	providers.ProviderPlanRequest,
) (providers.ProviderPlan, error) {
	return providers.ProviderPlan{
		Actions:     []model.PlanAction{},
		Diagnostics: []model.Diagnostic{},
	}, nil
}

func (provider *fakeProvider) Apply(
	context.Context,
	providers.ProviderAction,
	providers.ActionRuntime,
) (providers.ActionResult, error) {
	return providers.ActionResult{
		Outputs:     []model.ActionOutput{},
		Diagnostics: []model.Diagnostic{},
	}, nil
}

func (provider *fakeProvider) ExecutionSpec(
	_ context.Context,
	request providers.ExecutionRequest,
) (providers.ExecutionSpec, error) {
	return providers.ExecutionSpec{
		Executable:       "/fixture/bin/" + request.Executable,
		Arguments:        append([]string(nil), request.Arguments...),
		WorkingDirectory: request.WorkingDirectory,
		Environment:      append([]string(nil), request.Environment...),
		InputMode:        providers.InputInherit,
		OutputMode:       providers.OutputStream,
	}, nil
}
