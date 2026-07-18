package app_test

import (
	"context"
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
		Executable:       request.Executable,
		Arguments:        request.Arguments,
		WorkingDirectory: request.WorkingDirectory,
		Environment:      request.Environment,
		InputMode:        providers.InputInherit,
		OutputMode:       providers.OutputStream,
	}, nil
}
