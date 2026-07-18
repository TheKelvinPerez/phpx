package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/discovery"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
)

func TestDoctorInspectsProvidersOfflineAndBuildsReadOnlyAnalysis(t *testing.T) {
	facts := compatibleFacts()
	nativeProvider := &fakeProvider{
		name: "native",
		observation: model.ProviderObservation{
			Provider:  "native",
			Available: true,
			Capabilities: []model.Capability{
				model.CapabilityExecuteCommand,
				model.CapabilityInspectRuntime,
			},
			Runtimes: []model.RuntimeObservation{
				{
					Name:    "php",
					Version: "8.5.0",
					SAPI:    "cli",
					Source: model.SourceReference{
						Path: "/fixture/bin/php",
						Kind: "provider_executable",
					},
				},
			},
			Composer: []model.ComposerObservation{
				{
					Version:  "2.9.5",
					Source:   "system",
					Path:     "/fixture/bin/composer",
					Identity: "sha256:composer",
				},
			},
			Fingerprint: "sha256:native",
		},
	}
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			_ context.Context,
			request discovery.Request,
		) (model.ProjectFacts, error) {
			if request.StartPath != "/workspace" {
				t.Fatalf("unexpected discovery request %#v", request)
			}

			return facts, nil
		},
		Providers: []providers.Provider{nativeProvider},
	})

	analysis, err := application.Doctor(t.Context(), app.DoctorRequest{
		ProjectPath: "/workspace",
		Provider:    "native",
	})
	if err != nil {
		t.Fatalf("doctor analysis: %v", err)
	}

	if len(nativeProvider.inspectRequests) != 1 {
		t.Fatalf(
			"expected one provider inspection, got %#v",
			nativeProvider.inspectRequests,
		)
	}
	inspection := nativeProvider.inspectRequests[0]
	if !inspection.Offline {
		t.Fatal("doctor provider inspection must prohibit network access")
	}
	if inspection.Project.IdentityKey != facts.Identity.IdentityKey {
		t.Fatalf("unexpected inspected project %#v", inspection.Project)
	}
	if analysis.Facts.Identity.IdentityKey != facts.Identity.IdentityKey {
		t.Fatalf("unexpected project facts %#v", analysis.Facts)
	}
	if len(analysis.Observations) != 1 ||
		analysis.Observations[0].Provider != "native" {
		t.Fatalf("unexpected provider observations %#v", analysis.Observations)
	}
	if analysis.Plan.Operation != model.OperationDoctor ||
		analysis.Plan.Provider.Name != "native" ||
		analysis.Plan.Provider.Reason != "explicit" {
		t.Fatalf("unexpected doctor plan %#v", analysis.Plan)
	}
	if len(analysis.Plan.Actions) != 0 {
		t.Fatalf("doctor must not plan actions, got %#v", analysis.Plan.Actions)
	}
}

func TestPlanReportsIncompatibleNativePHPWithoutProviderMutation(t *testing.T) {
	nativeProvider := &fakeProvider{
		name: "native",
		observation: model.ProviderObservation{
			Provider:  "native",
			Available: true,
			Capabilities: []model.Capability{
				model.CapabilityExecuteCommand,
				model.CapabilityInspectRuntime,
			},
			Runtimes: []model.RuntimeObservation{
				{
					Name:    "php",
					Version: "8.3.0",
					SAPI:    "cli",
					Source: model.SourceReference{
						Path: "/fixture/bin/php",
						Kind: "provider_executable",
					},
				},
			},
			Composer: []model.ComposerObservation{
				{
					Version:  "2.9.5",
					Source:   "system",
					Path:     "/fixture/bin/composer",
					Identity: "sha256:composer",
				},
			},
			Fingerprint: "sha256:native",
		},
	}
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			context.Context,
			discovery.Request,
		) (model.ProjectFacts, error) {
			return compatibleFacts(), nil
		},
		Providers: []providers.Provider{nativeProvider},
	})

	analysis, err := application.Plan(t.Context(), app.PlanRequest{
		ProjectPath: "/workspace",
		Provider:    "native",
		Offline:     true,
		Frozen:      true,
	})
	if err != nil {
		t.Fatalf("build native plan: %v", err)
	}

	if analysis.Plan.Operation != model.OperationSync {
		t.Fatalf("expected sync operation, got %q", analysis.Plan.Operation)
	}
	if !analysis.Plan.Policy.Offline || !analysis.Plan.Policy.Frozen {
		t.Fatalf("expected requested plan policy, got %#v", analysis.Plan.Policy)
	}
	php := requirementByName(t, analysis.Plan.Requirements, "php")
	if php.Status != model.ResolutionBlocked {
		t.Fatalf("expected incompatible PHP to block, got %#v", php)
	}
	diagnosticByCode(
		t,
		analysis.Plan.Diagnostics,
		"ELEFANTE_REQUIREMENT_INCOMPATIBLE",
	)
	if len(analysis.Plan.Actions) != 0 {
		t.Fatalf(
			"native plan must not relink or install PHP, got %#v",
			analysis.Plan.Actions,
		)
	}
}

func TestPlanIncludesSelectedProviderPreparation(t *testing.T) {
	ddevProvider := &fakeProvider{
		name: "ddev",
		observation: model.ProviderObservation{
			Provider:  "ddev",
			Available: true,
			State:     model.ProviderStateStopped,
			Runtimes: []model.RuntimeObservation{
				{
					Name:    "php",
					Version: "8.4.3",
					Source: model.SourceReference{
						Path: "/workspace/.ddev/config.yaml",
						Kind: "provider_config",
					},
				},
			},
			Extensions: []model.ExtensionObservation{
				{
					Name:      "ext-json",
					Version:   "8.4.3",
					Available: true,
					Source: model.SourceReference{
						Path: "/workspace/.ddev/config.yaml",
						Kind: "provider_config",
					},
				},
			},
			Composer: []model.ComposerObservation{
				{
					Version:  "2.9.5",
					Source:   "ddev",
					Path:     "/usr/local/bin/composer",
					Identity: "sha256:composer",
					Reference: model.SourceReference{
						Path:  "/usr/local/bin/composer",
						Kind:  "provider_executable",
						Field: "example:web",
					},
				},
			},
			Fingerprint: "sha256:ddev",
		},
		planResult: providers.ProviderPlan{
			Actions: []model.PlanAction{
				{
					Kind:       model.ActionPrepareProvider,
					Summary:    "Start the DDEV project environment.",
					Effect:     model.EffectProviderMutation,
					Network:    model.NetworkNone,
					Trust:      model.TrustNone,
					Reversible: true,
					Inputs: []model.ActionInput{
						{Name: "operation", Value: "start"},
					},
				},
			},
		},
	}
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			context.Context,
			discovery.Request,
		) (model.ProjectFacts, error) {
			return compatibleFacts(), nil
		},
		Providers: []providers.Provider{ddevProvider},
	})

	analysis, err := application.Plan(t.Context(), app.PlanRequest{
		ProjectPath: "/workspace",
		Provider:    "ddev",
	})
	if err != nil {
		t.Fatalf("build DDEV plan: %v", err)
	}

	if len(ddevProvider.planRequests) != 1 {
		t.Fatalf("expected one provider plan request, got %#v", ddevProvider.planRequests)
	}
	planRequest := ddevProvider.planRequests[0]
	if planRequest.Facts.Identity.IdentityKey != "sha256:project" ||
		planRequest.Observation.Provider != "ddev" ||
		planRequest.Policy.Offline ||
		planRequest.Policy.Frozen {
		t.Fatalf("unexpected provider plan request %#v", planRequest)
	}
	if len(analysis.Plan.Actions) != 4 ||
		analysis.Plan.Actions[0].Kind != model.ActionPrepareProvider {
		t.Fatalf("provider preparation did not reach canonical plan: %#v", analysis.Plan.Actions)
	}
}

func TestPlanDoesNotRequestActionsFromUnselectedProviders(t *testing.T) {
	ddevProvider := &fakeProvider{
		name:        "ddev",
		observation: compatibleProviderObservation("ddev"),
	}
	nativeProvider := &fakeProvider{
		name:        "native",
		observation: compatibleProviderObservation("native"),
		planError:   errors.New("unselected provider plan failed"),
	}
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			context.Context,
			discovery.Request,
		) (model.ProjectFacts, error) {
			return compatibleFacts(), nil
		},
		Providers: []providers.Provider{
			nativeProvider,
			ddevProvider,
		},
	})

	analysis, err := application.Plan(t.Context(), app.PlanRequest{
		ProjectPath: "/workspace",
		Provider:    "ddev",
	})
	if err != nil {
		t.Fatalf("build explicitly selected DDEV plan: %v", err)
	}

	if analysis.Plan.Provider.Name != "ddev" {
		t.Fatalf("expected DDEV selection, got %#v", analysis.Plan.Provider)
	}
	if len(ddevProvider.planRequests) != 1 {
		t.Fatalf("selected provider was not planned once: %#v", ddevProvider.planRequests)
	}
	if len(nativeProvider.planRequests) != 0 {
		t.Fatalf("unselected provider was asked to plan: %#v", nativeProvider.planRequests)
	}
}

func compatibleProviderObservation(name string) model.ProviderObservation {
	return model.ProviderObservation{
		Provider:  name,
		Available: true,
		Runtimes: []model.RuntimeObservation{
			{
				Name:    "php",
				Version: "8.4.3",
				Source: model.SourceReference{
					Path: "/fixture/bin/php",
					Kind: "provider_executable",
				},
			},
		},
		Composer: []model.ComposerObservation{
			{
				Version:  "2.9.5",
				Source:   name,
				Path:     "/fixture/bin/composer",
				Identity: "sha256:composer-" + name,
			},
		},
		Fingerprint: "sha256:" + name,
	}
}

func compatibleFacts() model.ProjectFacts {
	return model.ProjectFacts{
		Identity: model.ProjectIdentity{
			ComposerRoot:    "/workspace",
			ApplicationRoot: "/workspace",
			WorkspaceRoot:   "/workspace",
			IdentityKey:     "sha256:project",
		},
		Composer: model.ComposerFacts{
			Manifest: model.ComposerManifestFacts{
				Path: "/workspace/composer.json",
				Name: "acme/example",
			},
			Lock: model.ComposerLockFacts{
				Path:   "/workspace/composer.lock",
				Status: model.ComposerLockFresh,
			},
			PlatformRequirements: []model.Requirement{
				{
					Name:       "php",
					Kind:       model.RequirementPHP,
					Constraint: "^8.4",
					Scope:      model.RequirementScopeRoot,
					Sources: []model.SourceReference{
						{
							Path:  "/workspace/composer.json",
							Kind:  "composer_manifest",
							Field: "/require/php",
						},
					},
				},
			},
		},
		InputFingerprints: []model.InputFingerprint{
			{
				Path:   "/workspace/composer.json",
				Kind:   "composer_manifest",
				SHA256: "manifest",
				Size:   128,
			},
		},
	}
}

func requirementByName(
	t *testing.T,
	requirements []model.RequirementResolution,
	name string,
) model.RequirementResolution {
	t.Helper()

	for _, requirement := range requirements {
		if requirement.Name == name {
			return requirement
		}
	}
	t.Fatalf("expected requirement %q, got %#v", name, requirements)

	return model.RequirementResolution{}
}

func diagnosticByCode(
	t *testing.T,
	diagnostics []model.Diagnostic,
	code string,
) model.Diagnostic {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return diagnostic
		}
	}
	t.Fatalf("expected diagnostic %q, got %#v", code, diagnostics)

	return model.Diagnostic{}
}

type fakeProvider struct {
	name            string
	observation     model.ProviderObservation
	inspectRequests []providers.InspectRequest
	planRequests    []providers.ProviderPlanRequest
	planResult      providers.ProviderPlan
	planError       error
}

func (provider *fakeProvider) Name() string {
	return provider.name
}

func (provider *fakeProvider) Inspect(
	_ context.Context,
	request providers.InspectRequest,
) (model.ProviderObservation, error) {
	provider.inspectRequests = append(provider.inspectRequests, request)

	return provider.observation, nil
}

func (provider *fakeProvider) Plan(
	_ context.Context,
	request providers.ProviderPlanRequest,
) (providers.ProviderPlan, error) {
	provider.planRequests = append(provider.planRequests, request)

	return provider.planResult, provider.planError
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
		Executable:       request.Executable,
		Arguments:        request.Arguments,
		WorkingDirectory: request.WorkingDirectory,
		Environment:      request.Environment,
		InputMode:        providers.InputInherit,
		OutputMode:       providers.OutputStream,
	}, nil
}
