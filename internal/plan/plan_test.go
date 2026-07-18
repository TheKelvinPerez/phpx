package plan_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/plan"
)

func TestBuildCompatiblePlan(t *testing.T) {
	request := compatibleRequest()

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if result.SchemaVersion != model.PlanSchemaVersion {
		t.Fatalf("expected schema %q, got %q", model.PlanSchemaVersion, result.SchemaVersion)
	}
	if result.Operation != model.OperationSync {
		t.Fatalf("expected sync operation, got %q", result.Operation)
	}
	if result.Provider.Name != "native" {
		t.Fatalf("expected native provider, got %#v", result.Provider)
	}
	if len(result.Requirements) != 2 {
		t.Fatalf("expected PHP and JSON resolutions, got %#v", result.Requirements)
	}
	for _, resolution := range result.Requirements {
		if resolution.Status != model.ResolutionSatisfied {
			t.Fatalf("expected satisfied resolution, got %#v", resolution)
		}
	}

	expectedActions := []model.ActionKind{
		model.ActionInstallDependencies,
		model.ActionVerifyPlatform,
		model.ActionRecordState,
	}
	if len(result.Actions) != len(expectedActions) {
		t.Fatalf("expected actions %#v, got %#v", expectedActions, result.Actions)
	}
	for index, expected := range expectedActions {
		if result.Actions[index].Kind != expected {
			t.Fatalf(
				"expected action %d to be %q, got %#v",
				index,
				expected,
				result.Actions[index],
			)
		}
		if result.Actions[index].ID == "" {
			t.Fatalf("expected deterministic action ID, got %#v", result.Actions[index])
		}
	}
	if !strings.HasPrefix(result.Digest, "sha256:") || len(result.Digest) != 71 {
		t.Fatalf("expected SHA256 plan digest, got %q", result.Digest)
	}
}

func TestBuildReportsConflictingRequirementSources(t *testing.T) {
	request := compatibleRequest()
	request.Facts.VersionFiles = []model.VersionFileFact{
		{
			Runtime: "php",
			Version: "8.3",
			Source: model.SourceReference{
				Path: "/workspace/.php-version",
				Kind: "php_version",
				Line: 1,
			},
		},
	}

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	php := requirementResolution(t, result, "php")
	if php.Status != model.ResolutionBlocked {
		t.Fatalf("expected blocked PHP resolution, got %#v", php)
	}
	diagnostic := diagnosticByCode(
		t,
		result.Diagnostics,
		"ELEFANTE_REQUIREMENT_CONFLICT",
	)
	if len(diagnostic.Sources) != 2 {
		t.Fatalf("expected Composer and version file sources, got %#v", diagnostic.Sources)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("expected no actions for conflicting intent, got %#v", result.Actions)
	}
}

func TestBuildPlansSupportedRuntimePreparation(t *testing.T) {
	request := compatibleRequest()
	request.Observations[0].Runtimes[0].Version = "8.2.29"
	request.Observations[0].Capabilities = []model.Capability{
		model.CapabilityInstallRuntime,
	}

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	php := requirementResolution(t, result, "php")
	if php.Status != model.ResolutionActionRequired {
		t.Fatalf("expected runtime action requirement, got %#v", php)
	}
	if php.SelectedValue != "8.5.0" {
		t.Fatalf("expected highest supported PHP 8 target, got %#v", php)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected an actionable plan, got %#v", result.Diagnostics)
	}
	if len(result.Actions) != 4 ||
		result.Actions[0].Kind != model.ActionPrepareRuntime {
		t.Fatalf("expected runtime preparation first, got %#v", result.Actions)
	}
}

func TestBuildKeepsMissingOptionalExtensionActionable(t *testing.T) {
	request := compatibleRequest()
	request.Facts.Configuration.Path = "/workspace/elefante.toml"
	request.Facts.Configuration.Extensions.Optional = []string{"ext-xdebug"}

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	xdebug := requirementResolution(t, result, "ext-xdebug")
	if xdebug.Status != model.ResolutionOptionalMissing {
		t.Fatalf("expected optional extension status, got %#v", xdebug)
	}
	diagnostic := diagnosticByCode(
		t,
		result.Diagnostics,
		"ELEFANTE_OPTIONAL_EXTENSION_MISSING",
	)
	if diagnostic.Severity != model.SeverityWarning {
		t.Fatalf("expected optional warning, got %#v", diagnostic)
	}
	if len(result.Actions) != 3 {
		t.Fatalf("expected the plan to remain actionable, got %#v", result.Actions)
	}
}

func TestBuildBlocksAnIncompatibleObservedRuntime(t *testing.T) {
	request := compatibleRequest()
	request.Observations[0].Runtimes[0].Version = "8.3.19"

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	php := requirementResolution(t, result, "php")
	if php.Status != model.ResolutionBlocked {
		t.Fatalf("expected blocked PHP resolution, got %#v", php)
	}
	diagnosticByCode(
		t,
		result.Diagnostics,
		"ELEFANTE_REQUIREMENT_INCOMPATIBLE",
	)
	if len(result.Actions) != 0 {
		t.Fatalf("expected no actions for a blocked plan, got %#v", result.Actions)
	}
}

func TestBuildBlocksAmbiguousProvidersDeterministically(t *testing.T) {
	request := compatibleRequest()
	request.Facts.Configuration.Providers.Allowed = nil
	ddev := request.Observations[0]
	ddev.Provider = "ddev"
	ddev.Fingerprint = "sha256:ddev"
	request.Observations = append(request.Observations, ddev)

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if result.Provider.Name != "" {
		t.Fatalf("expected no arbitrary provider selection, got %#v", result.Provider)
	}
	diagnostic := diagnosticByCode(
		t,
		result.Diagnostics,
		"ELEFANTE_PROVIDER_AMBIGUOUS",
	)
	if len(diagnostic.Sources) != 2 ||
		diagnostic.Sources[0].Path != "ddev" ||
		diagnostic.Sources[1].Path != "native" {
		t.Fatalf("expected sorted provider sources, got %#v", diagnostic.Sources)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("expected no actions for ambiguous providers, got %#v", result.Actions)
	}
	for _, resolution := range result.Requirements {
		if resolution.Status != model.ResolutionAmbiguous {
			t.Fatalf("expected ambiguous resolution, got %#v", resolution)
		}
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "ELEFANTE_REQUIREMENT_UNAVAILABLE" {
			t.Fatalf("provider ambiguity should not report missing requirements: %#v", result.Diagnostics)
		}
	}
}

func TestBuildAppliesProviderSelectionPrecedence(t *testing.T) {
	request := compatibleRequest()
	request.Facts.Configuration.Providers.Allowed = nil
	ddev := request.Observations[0]
	ddev.Provider = "ddev"
	ddev.Fingerprint = "sha256:ddev"
	homebrew := request.Observations[0]
	homebrew.Provider = "homebrew"
	homebrew.Fingerprint = "sha256:homebrew"
	request.Observations = append(request.Observations, ddev, homebrew)
	request.Provider = "native"
	request.Facts.Configuration.Providers.Preferred = []string{"homebrew"}
	request.Facts.ProviderMarkers = []model.ProviderMarkerFact{
		{Provider: "ddev"},
	}
	request.PreviousProvider = "ddev"
	request.DefaultProvider = "homebrew"

	cases := []struct {
		name     string
		mutate   func(*plan.Request)
		provider string
		reason   string
	}{
		{
			name:     "explicit",
			mutate:   func(*plan.Request) {},
			provider: "native",
			reason:   "explicit",
		},
		{
			name: "configuration",
			mutate: func(request *plan.Request) {
				request.Provider = ""
			},
			provider: "homebrew",
			reason:   "configuration",
		},
		{
			name: "marker",
			mutate: func(request *plan.Request) {
				request.Provider = ""
				request.Facts.Configuration.Providers.Preferred = nil
			},
			provider: "ddev",
			reason:   "provider_marker",
		},
		{
			name: "workspace state",
			mutate: func(request *plan.Request) {
				request.Provider = ""
				request.Facts.Configuration.Providers.Preferred = nil
				request.Facts.ProviderMarkers = nil
			},
			provider: "ddev",
			reason:   "workspace_state",
		},
		{
			name: "user default",
			mutate: func(request *plan.Request) {
				request.Provider = ""
				request.Facts.Configuration.Providers.Preferred = nil
				request.Facts.ProviderMarkers = nil
				request.PreviousProvider = ""
			},
			provider: "homebrew",
			reason:   "user_default",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			candidate := request
			candidate.Facts.Configuration.Providers.Preferred = append(
				[]string(nil),
				request.Facts.Configuration.Providers.Preferred...,
			)
			candidate.Facts.ProviderMarkers = append(
				[]model.ProviderMarkerFact(nil),
				request.Facts.ProviderMarkers...,
			)
			testCase.mutate(&candidate)

			result, err := plan.Build(candidate)
			if err != nil {
				t.Fatalf("build plan: %v", err)
			}
			if result.Provider.Name != testCase.provider ||
				result.Provider.Reason != testCase.reason {
				t.Fatalf(
					"expected %s selection %q, got %#v",
					testCase.reason,
					testCase.provider,
					result.Provider,
				)
			}
		})
	}
}

func TestBuildSelectsTheOnlyCompatibleProvider(t *testing.T) {
	request := compatibleRequest()
	request.Facts.Configuration.Providers.Allowed = nil
	request.Observations[0].Runtimes[0].Version = "8.3.19"
	ddev := request.Observations[0]
	ddev.Provider = "ddev"
	ddev.Fingerprint = "sha256:ddev"
	ddev.Runtimes = []model.RuntimeObservation{
		{Name: "php", Version: "8.4.3"},
	}
	request.Observations = append(request.Observations, ddev)

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if result.Provider.Name != "ddev" ||
		result.Provider.Reason != "best_compatible" {
		t.Fatalf("expected the compatible provider, got %#v", result.Provider)
	}
	if len(result.Actions) != 3 {
		t.Fatalf("expected a compatible actionable plan, got %#v", result)
	}
}

func TestBuildReportsLegacyPHPWithoutBlockingExecution(t *testing.T) {
	request := compatibleRequest()
	request.Facts.Composer.PlatformRequirements[0].Constraint = "^8.2"
	request.Observations[0].Runtimes[0].Version = "8.2.29"

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	php := requirementResolution(t, result, "php")
	if php.Status != model.ResolutionLegacy {
		t.Fatalf("expected legacy PHP resolution, got %#v", php)
	}
	diagnostic := diagnosticByCode(
		t,
		result.Diagnostics,
		"ELEFANTE_LEGACY_RUNTIME",
	)
	if diagnostic.Severity != model.SeverityWarning {
		t.Fatalf("expected a legacy warning, got %#v", diagnostic)
	}
	if len(result.Actions) != 3 {
		t.Fatalf("expected legacy execution to remain actionable, got %#v", result.Actions)
	}
}

func TestBuildReportsLegacyLaravelWithoutBlockingExecution(t *testing.T) {
	request := compatibleRequest()
	laravelSource := model.SourceReference{
		Path:  "/workspace/composer.json",
		Kind:  "composer_manifest",
		Field: "/require/laravel~1framework",
	}
	request.Facts.Composer.Manifest.Requirements = []model.ComposerLink{
		{
			Name:       "laravel/framework",
			Constraint: "^11.0",
			Source:     laravelSource,
		},
	}
	request.Facts.Frameworks = []model.FrameworkFact{
		{
			Kind:       model.FrameworkLaravelApplication,
			Confidence: model.FrameworkConfidenceHigh,
			Primary:    true,
			Evidence: []model.FrameworkEvidence{
				{
					Kind:   "composer_requirement",
					Source: laravelSource,
				},
			},
		},
	}

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	diagnostic := diagnosticByCode(
		t,
		result.Diagnostics,
		"ELEFANTE_LEGACY_FRAMEWORK",
	)
	if diagnostic.Severity != model.SeverityWarning {
		t.Fatalf("expected a legacy framework warning, got %#v", diagnostic)
	}
	if len(result.Actions) != 3 {
		t.Fatalf("expected legacy framework execution to remain actionable, got %#v", result.Actions)
	}
}

func TestBuildBlocksNetworkRequiredActionsOffline(t *testing.T) {
	request := compatibleRequest()
	request.Policy.Offline = true

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	diagnostic := diagnosticByCode(
		t,
		result.Diagnostics,
		"ELEFANTE_OFFLINE_NETWORK_REQUIRED",
	)
	if diagnostic.Severity != model.SeverityError {
		t.Fatalf("expected an offline blocker, got %#v", diagnostic)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("expected no actions when offline inputs are missing, got %#v", result.Actions)
	}
}

func TestBuildBlocksMissingComposerLockInFrozenMode(t *testing.T) {
	request := compatibleRequest()
	request.Policy.Frozen = true
	request.Facts.Composer.Lock.Status = model.ComposerLockMissing
	request.Facts.Composer.Lock.Path = ""

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	diagnostic := diagnosticByCode(
		t,
		result.Diagnostics,
		"ELEFANTE_FROZEN_LOCK_REQUIRED",
	)
	if diagnostic.Severity != model.SeverityError {
		t.Fatalf("expected a frozen lock blocker, got %#v", diagnostic)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("expected no frozen lock mutation actions, got %#v", result.Actions)
	}
}

func TestBuildOrdersActionsByExecutionPhase(t *testing.T) {
	request := compatibleRequest()
	request.Observations[0].Runtimes[0].Version = "8.2.29"
	request.Observations[0].Capabilities = []model.Capability{
		model.CapabilityInstallExtension,
		model.CapabilityInstallRuntime,
	}
	request.Facts.Composer.PlatformRequirements = append(
		request.Facts.Composer.PlatformRequirements,
		model.Requirement{
			Name:       "ext-zeta",
			Kind:       model.RequirementExtension,
			Constraint: "*",
			Scope:      model.RequirementScopeRoot,
			Sources: []model.SourceReference{
				{
					Path:  "/workspace/composer.json",
					Kind:  "composer_manifest",
					Field: "/require/ext-zeta",
				},
			},
		},
		model.Requirement{
			Name:       "ext-alpha",
			Kind:       model.RequirementExtension,
			Constraint: "*",
			Scope:      model.RequirementScopeRoot,
			Sources: []model.SourceReference{
				{
					Path:  "/workspace/composer.json",
					Kind:  "composer_manifest",
					Field: "/require/ext-alpha",
				},
			},
		},
	)

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	expected := []model.ActionKind{
		model.ActionPrepareRuntime,
		model.ActionPrepareExtension,
		model.ActionPrepareExtension,
		model.ActionInstallDependencies,
		model.ActionVerifyPlatform,
		model.ActionRecordState,
	}
	if len(result.Actions) != len(expected) {
		t.Fatalf("expected action order %#v, got %#v", expected, result.Actions)
	}
	for index, kind := range expected {
		if result.Actions[index].Kind != kind {
			t.Fatalf("expected action %d to be %q, got %#v", index, kind, result.Actions)
		}
		if index == 0 {
			continue
		}
		if len(result.Actions[index].Dependencies) != 1 ||
			result.Actions[index].Dependencies[0] != result.Actions[index-1].ID {
			t.Fatalf("expected a stable action chain at %d, got %#v", index, result.Actions)
		}
	}
	if result.Actions[1].Inputs[0].Value != "ext-alpha" ||
		result.Actions[2].Inputs[0].Value != "ext-zeta" {
		t.Fatalf("expected extensions ordered by name, got %#v", result.Actions)
	}
}

func TestBuildDeclaresComposerTrustRequirements(t *testing.T) {
	request := compatibleRequest()
	request.Facts.Composer.Plugins = []model.ComposerPlugin{
		{
			Name:    "acme/plugin",
			Version: "1.2.3",
			Source: model.SourceReference{
				Path:  "/workspace/composer.lock",
				Kind:  "composer_lock",
				Field: "/packages/0",
			},
		},
	}
	request.Facts.Composer.Scripts = []model.ComposerScript{
		{
			Name:          "post-install-cmd",
			CommandCount:  1,
			ContentSHA256: "sha256:script",
			Source: model.SourceReference{
				Path:  "/workspace/composer.json",
				Kind:  "composer_manifest",
				Field: "/scripts/post-install-cmd",
			},
		},
	}

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if len(result.Trust) != 2 {
		t.Fatalf("expected plugin and script trust requirements, got %#v", result.Trust)
	}
	if result.Trust[0].Class != model.TrustComposerPlugins ||
		result.Trust[1].Class != model.TrustComposerScripts {
		t.Fatalf("expected stable trust order, got %#v", result.Trust)
	}
	for _, requirement := range result.Trust {
		if !strings.HasPrefix(requirement.Fingerprint, "sha256:") {
			t.Fatalf("expected content addressed trust, got %#v", requirement)
		}
	}
	install := actionByKind(t, result, model.ActionInstallDependencies)
	if install.Trust != model.TrustComposerPlugins {
		t.Fatalf("expected install action to declare plugin trust, got %#v", install)
	}
}

func TestBuildTrustFingerprintTracksComposerExecutionInputs(t *testing.T) {
	first := compatibleRequest()
	first.Facts.Composer.Scripts = []model.ComposerScript{
		{
			Name:          "post-install-cmd",
			CommandCount:  1,
			ContentSHA256: "sha256:first",
		},
	}
	second := compatibleRequest()
	second.Facts.Composer.Scripts = []model.ComposerScript{
		{
			Name:          "post-install-cmd",
			CommandCount:  1,
			ContentSHA256: "sha256:second",
		},
	}

	firstPlan, err := plan.Build(first)
	if err != nil {
		t.Fatalf("build first plan: %v", err)
	}
	secondPlan, err := plan.Build(second)
	if err != nil {
		t.Fatalf("build second plan: %v", err)
	}

	if firstPlan.Trust[0].Fingerprint == secondPlan.Trust[0].Fingerprint {
		t.Fatalf("expected script content to invalidate Composer trust")
	}
	if firstPlan.Digest == secondPlan.Digest {
		t.Fatalf("expected trust change to alter plan digest")
	}
}

func TestBuildDigestExcludesDiagnosticDisplayWording(t *testing.T) {
	first := compatibleRequest()
	first.Facts.Diagnostics = []model.Diagnostic{
		{
			Code:     "ELEFANTE_DISPLAY_ONLY",
			Severity: model.SeverityWarning,
			Message:  "Alpha wording.",
			Sources: []model.SourceReference{
				{Path: "/workspace/a", Kind: "fixture"},
			},
		},
		{
			Code:     "ELEFANTE_DISPLAY_ONLY",
			Severity: model.SeverityWarning,
			Message:  "Zulu wording.",
			Sources: []model.SourceReference{
				{Path: "/workspace/b", Kind: "fixture"},
			},
		},
	}
	second := compatibleRequest()
	second.Facts.Diagnostics = []model.Diagnostic{
		{
			Code:     "ELEFANTE_DISPLAY_ONLY",
			Severity: model.SeverityWarning,
			Message:  "Zulu wording.",
			Sources: []model.SourceReference{
				{Path: "/workspace/a", Kind: "fixture"},
			},
		},
		{
			Code:     "ELEFANTE_DISPLAY_ONLY",
			Severity: model.SeverityWarning,
			Message:  "Alpha wording.",
			Sources: []model.SourceReference{
				{Path: "/workspace/b", Kind: "fixture"},
			},
		},
	}

	firstPlan, err := plan.Build(first)
	if err != nil {
		t.Fatalf("build first plan: %v", err)
	}
	secondPlan, err := plan.Build(second)
	if err != nil {
		t.Fatalf("build second plan: %v", err)
	}

	if firstPlan.Digest != secondPlan.Digest {
		t.Fatalf(
			"expected display wording independent digest, got %q and %q",
			firstPlan.Digest,
			secondPlan.Digest,
		)
	}
}

func TestBuildDigestChangesForRelevantInputs(t *testing.T) {
	baseline, err := plan.Build(compatibleRequest())
	if err != nil {
		t.Fatalf("build baseline plan: %v", err)
	}

	cases := map[string]func(*plan.Request){
		"project fingerprint": func(request *plan.Request) {
			request.Facts.InputFingerprints[0].SHA256 = "changed"
		},
		"provider fingerprint": func(request *plan.Request) {
			request.Observations[0].Fingerprint = "sha256:changed"
		},
		"runtime identity": func(request *plan.Request) {
			request.Observations[0].Runtimes[0].Version = "8.4.4"
		},
		"extension identity": func(request *plan.Request) {
			request.Observations[0].Extensions[0].Version = "8.4.4"
		},
		"composer identity": func(request *plan.Request) {
			request.Observations[0].Composer[0].Identity = "sha256:changed"
		},
		"project identity": func(request *plan.Request) {
			request.Facts.Identity.IdentityKey = "sha256:changed"
		},
		"offline policy": func(request *plan.Request) {
			request.Policy.Offline = true
		},
		"frozen policy": func(request *plan.Request) {
			request.Policy.Frozen = true
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			request := compatibleRequest()
			mutate(&request)
			changed, err := plan.Build(request)
			if err != nil {
				t.Fatalf("build changed plan: %v", err)
			}
			if changed.Digest == baseline.Digest {
				t.Fatalf("expected %s to alter digest %q", name, baseline.Digest)
			}
		})
	}
}

func TestBuildCanonicalizesEquivalentUnorderedFacts(t *testing.T) {
	first := compatibleRequest()
	first.Provider = "native"
	first.Facts.InputFingerprints = append(
		first.Facts.InputFingerprints,
		model.InputFingerprint{
			Path:   "/workspace/elefante.toml",
			Kind:   "elefante_config",
			SHA256: "config-digest",
			Size:   32,
		},
	)
	first.Observations[0].Capabilities = []model.Capability{
		model.CapabilityInspectComposer,
		model.CapabilityInspectRuntime,
	}
	ddev := first.Observations[0]
	ddev.Provider = "ddev"
	ddev.Fingerprint = "sha256:ddev"
	first.Observations = append(first.Observations, ddev)

	second := compatibleRequest()
	second.Provider = "native"
	second.Facts.Composer.PlatformRequirements[0],
		second.Facts.Composer.PlatformRequirements[1] =
		second.Facts.Composer.PlatformRequirements[1],
		second.Facts.Composer.PlatformRequirements[0]
	second.Facts.InputFingerprints = []model.InputFingerprint{
		{
			Path:   "/workspace/elefante.toml",
			Kind:   "elefante_config",
			SHA256: "config-digest",
			Size:   32,
		},
		second.Facts.InputFingerprints[0],
	}
	second.Observations[0].Capabilities = []model.Capability{
		model.CapabilityInspectRuntime,
		model.CapabilityInspectComposer,
	}
	ddev = second.Observations[0]
	ddev.Provider = "ddev"
	ddev.Fingerprint = "sha256:ddev"
	second.Observations = []model.ProviderObservation{
		ddev,
		second.Observations[0],
	}

	firstPlan, err := plan.Build(first)
	if err != nil {
		t.Fatalf("build first plan: %v", err)
	}
	secondPlan, err := plan.Build(second)
	if err != nil {
		t.Fatalf("build second plan: %v", err)
	}
	firstJSON, err := json.Marshal(firstPlan)
	if err != nil {
		t.Fatalf("encode first plan: %v", err)
	}
	secondJSON, err := json.Marshal(secondPlan)
	if err != nil {
		t.Fatalf("encode second plan: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf(
			"expected byte identical canonical plans\nfirst:  %s\nsecond: %s",
			firstJSON,
			secondJSON,
		)
	}
}

func TestBuildDoesNotMutateRequest(t *testing.T) {
	request := compatibleRequest()
	request.Facts.Diagnostics = []model.Diagnostic{
		{
			Code:     "ELEFANTE_FIXTURE_WARNING",
			Severity: model.SeverityWarning,
			Message:  "Fixture warning.",
			Sources: []model.SourceReference{
				{Path: "/workspace/z", Kind: "fixture"},
				{Path: "/workspace/a", Kind: "fixture"},
			},
		},
	}
	before, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("encode request before planning: %v", err)
	}

	if _, err := plan.Build(request); err != nil {
		t.Fatalf("build plan: %v", err)
	}

	after, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("encode request after planning: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf(
			"planner mutated its request\nbefore: %s\nafter:  %s",
			before,
			after,
		)
	}
}

func TestBuildResolvesCommittedComposerPolicy(t *testing.T) {
	request := compatibleRequest()
	request.Facts.Configuration.Path = "/workspace/elefante.toml"
	request.Facts.Configuration.Composer.Constraint = "^2.8"

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	composer := requirementResolution(t, result, "composer")
	if composer.Status != model.ResolutionSatisfied ||
		composer.SelectedValue != "2.8.9" {
		t.Fatalf("expected selected Composer to satisfy policy, got %#v", composer)
	}
	if len(composer.Sources) != 1 ||
		composer.Sources[0].Path != "/workspace/elefante.toml" {
		t.Fatalf("expected committed policy source, got %#v", composer.Sources)
	}
}

func TestBuildSelectsComposerByConstraintThenSourcePrecedence(t *testing.T) {
	tests := []struct {
		name      string
		composers []model.ComposerObservation
		identity  string
	}{
		{
			name: "managed before system",
			composers: []model.ComposerObservation{
				{
					Version:  "2.8.9",
					Source:   "system",
					Identity: "sha256:system",
				},
				{
					Version:  "2.8.8",
					Source:   "managed",
					Identity: "sha256:managed",
				},
			},
			identity: "sha256:managed",
		},
		{
			name: "compatible system before incompatible managed",
			composers: []model.ComposerObservation{
				{
					Version:  "2.7.9",
					Source:   "managed",
					Identity: "sha256:managed",
				},
				{
					Version:  "2.8.9",
					Source:   "system",
					Identity: "sha256:system",
				},
			},
			identity: "sha256:system",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := compatibleRequest()
			request.Facts.Configuration.Path = "/workspace/elefante.toml"
			request.Facts.Configuration.Composer.Constraint = "^2.8"
			request.Observations[0].Composer = testCase.composers

			result, err := plan.Build(request)
			if err != nil {
				t.Fatalf("build plan: %v", err)
			}

			install := actionByKind(
				t,
				result,
				model.ActionInstallDependencies,
			)
			if value := actionInputValue(
				t,
				install,
				"composer",
			); value != testCase.identity {
				t.Fatalf(
					"expected Composer identity %q, got %#v",
					testCase.identity,
					install,
				)
			}
		})
	}
}

func TestBuildReportsInvalidRequirementConstraint(t *testing.T) {
	request := compatibleRequest()
	request.Facts.Composer.PlatformRequirements[0].Constraint = "not a constraint"

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("expected a diagnostic plan, got error: %v", err)
	}

	php := requirementResolution(t, result, "php")
	if php.Status != model.ResolutionBlocked {
		t.Fatalf("expected invalid PHP requirement to block, got %#v", php)
	}
	diagnostic := diagnosticByCode(
		t,
		result.Diagnostics,
		"ELEFANTE_REQUIREMENT_INVALID",
	)
	if len(diagnostic.Sources) != 1 {
		t.Fatalf("expected attributed invalid requirement, got %#v", diagnostic)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("expected no actions for invalid intent, got %#v", result.Actions)
	}
}

func compatibleRequest() plan.Request {
	composerPath := "/workspace/composer.json"

	return plan.Request{
		Operation: model.OperationSync,
		Facts: model.ProjectFacts{
			Identity: model.ProjectIdentity{
				RepositoryRoot:  "/workspace",
				ComposerRoot:    "/workspace",
				ApplicationRoot: "/workspace",
				WorkspaceRoot:   "/workspace",
				IdentityKey:     "sha256:project",
			},
			Composer: model.ComposerFacts{
				Manifest: model.ComposerManifestFacts{
					Path: composerPath,
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
								Path:  composerPath,
								Kind:  "composer_manifest",
								Field: "/require/php",
							},
						},
					},
					{
						Name:       "ext-json",
						Kind:       model.RequirementExtension,
						Constraint: "*",
						Scope:      model.RequirementScopeRoot,
						Sources: []model.SourceReference{
							{
								Path:  composerPath,
								Kind:  "composer_manifest",
								Field: "/require/ext-json",
							},
						},
					},
				},
			},
			Configuration: model.ConfigFacts{
				Providers: model.ConfigProviderPolicy{
					Allowed: []string{"native"},
				},
			},
			InputFingerprints: []model.InputFingerprint{
				{
					Path:   composerPath,
					Kind:   "composer_manifest",
					SHA256: "manifest-digest",
					Size:   128,
				},
			},
		},
		Observations: []model.ProviderObservation{
			{
				Provider:  "native",
				Available: true,
				Runtimes: []model.RuntimeObservation{
					{
						Name:    "php",
						Version: "8.4.3",
					},
				},
				Extensions: []model.ExtensionObservation{
					{
						Name:      "ext-json",
						Version:   "8.4.3",
						Available: true,
					},
				},
				Composer: []model.ComposerObservation{
					{
						Version:  "2.8.9",
						Source:   "system",
						Identity: "sha256:composer",
					},
				},
				Fingerprint: "sha256:native",
			},
		},
	}
}

func TestDoctorPlanExplainsSelectionWithoutMutationActions(t *testing.T) {
	request := compatibleRequest()
	request.Operation = model.OperationDoctor

	result, err := plan.Build(request)
	if err != nil {
		t.Fatalf("build doctor plan: %v", err)
	}

	if result.Operation != model.OperationDoctor {
		t.Fatalf("expected doctor operation, got %q", result.Operation)
	}
	if result.Provider.Name != "native" ||
		result.Provider.Reason != "only_available" {
		t.Fatalf("expected explained native selection, got %#v", result.Provider)
	}
	if len(result.Requirements) == 0 {
		t.Fatal("expected doctor to resolve project requirements")
	}
	if len(result.Actions) != 0 {
		t.Fatalf("doctor must remain action free, got %#v", result.Actions)
	}
}

func actionByKind(
	t *testing.T,
	result model.Plan,
	kind model.ActionKind,
) model.PlanAction {
	t.Helper()

	for _, action := range result.Actions {
		if action.Kind == kind {
			return action
		}
	}
	t.Fatalf("expected action %q, got %#v", kind, result.Actions)

	return model.PlanAction{}
}

func actionInputValue(
	t *testing.T,
	action model.PlanAction,
	name string,
) string {
	t.Helper()

	for _, input := range action.Inputs {
		if input.Name == name {
			return input.Value
		}
	}
	t.Fatalf("expected action input %q, got %#v", name, action.Inputs)

	return ""
}

func requirementResolution(
	t *testing.T,
	result model.Plan,
	name string,
) model.RequirementResolution {
	t.Helper()

	for _, resolution := range result.Requirements {
		if resolution.Name == name {
			return resolution
		}
	}
	t.Fatalf("expected requirement %q, got %#v", name, result.Requirements)

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
