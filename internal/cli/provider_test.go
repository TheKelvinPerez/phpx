package cli_test

import (
	"context"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
)

func testProviders() []providers.Provider {
	return providerSet(&testProvider{
		observation: model.ProviderObservation{
			Provider:     "native",
			Available:    true,
			Capabilities: []model.Capability{},
			Runtimes:     []model.RuntimeObservation{},
			Composer:     []model.ComposerObservation{},
			Extensions:   []model.ExtensionObservation{},
			Diagnostics:  []model.Diagnostic{},
			Fingerprint:  "sha256:native",
		},
	})
}

func providerSet(provider providers.Provider) []providers.Provider {
	return []providers.Provider{provider}
}

type testProvider struct {
	observation     model.ProviderObservation
	inspectRequests []providers.InspectRequest
}

func (provider *testProvider) Name() string {
	return provider.observation.Provider
}

func (provider *testProvider) Inspect(
	_ context.Context,
	request providers.InspectRequest,
) (model.ProviderObservation, error) {
	provider.inspectRequests = append(provider.inspectRequests, request)

	return provider.observation, nil
}

func (provider *testProvider) Plan(
	context.Context,
	providers.ProviderPlanRequest,
) (providers.ProviderPlan, error) {
	return providers.ProviderPlan{
		Actions:     []model.PlanAction{},
		Diagnostics: []model.Diagnostic{},
	}, nil
}

func (provider *testProvider) Apply(
	context.Context,
	providers.ProviderAction,
	providers.ActionRuntime,
) (providers.ActionResult, error) {
	return providers.ActionResult{
		Outputs:     []model.ActionOutput{},
		Diagnostics: []model.Diagnostic{},
	}, nil
}

func (provider *testProvider) ExecutionSpec(
	_ context.Context,
	request providers.ExecutionRequest,
) (providers.ExecutionSpec, error) {
	return providers.ExecutionSpec{
		Executable:       request.Executable,
		Arguments:        append([]string(nil), request.Arguments...),
		WorkingDirectory: request.WorkingDirectory,
		Environment:      append([]string(nil), request.Environment...),
		InputMode:        providers.InputInherit,
		OutputMode:       providers.OutputStream,
	}, nil
}
