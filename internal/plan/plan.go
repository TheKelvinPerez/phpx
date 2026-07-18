package plan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/elefantephp/elefante/internal/constraints"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
)

type Request struct {
	Operation        model.Operation
	Facts            model.ProjectFacts
	Observations     []model.ProviderObservation
	Provider         string
	PreviousProvider string
	DefaultProvider  string
	Policy           model.PlanPolicy
	ProviderPlans    map[string]providers.ProviderPlan
}

type requirementGroup struct {
	name         string
	kind         model.RequirementKind
	requirements []model.Requirement
	optional     bool
}

func Build(request Request) (model.Plan, error) {
	operation := request.Operation
	if operation == "" {
		operation = model.OperationSync
	}

	selection, observation, providerDiagnostics := selectProvider(request)
	providerPlan := selectedProviderPlan(request.ProviderPlans, selection.Name)
	resolutions, requirementDiagnostics, err := resolveRequirements(
		request.Facts,
		observation,
	)
	if err != nil {
		return model.Plan{}, err
	}
	if diagnosticsContain(
		providerDiagnostics,
		"ELEFANTE_PROVIDER_AMBIGUOUS",
	) {
		resolutions, requirementDiagnostics = ambiguousRequirements(
			resolutions,
			requirementDiagnostics,
		)
	}

	diagnostics := cloneDiagnostics(request.Facts.Diagnostics)
	diagnostics = append(diagnostics, providerDiagnostics...)
	diagnostics = append(
		diagnostics,
		cloneDiagnostics(providerPlan.Diagnostics)...,
	)
	diagnostics = append(diagnostics, requirementDiagnostics...)
	diagnostics = append(
		diagnostics,
		legacyFrameworkDiagnostics(request.Facts)...,
	)
	if observation != nil {
		diagnostics = append(
			diagnostics,
			cloneDiagnostics(observation.Diagnostics)...,
		)
	}
	sortDiagnostics(diagnostics)

	result := model.Plan{
		SchemaVersion: model.PlanSchemaVersion,
		Operation:     operation,
		Project:       request.Facts.Identity,
		Provider:      selection,
		Requirements:  resolutions,
		Actions:       make([]model.PlanAction, 0),
		Diagnostics:   diagnostics,
		Inputs:        cloneFingerprints(request.Facts.InputFingerprints),
		Policy:        request.Policy,
	}
	sortFingerprints(result.Inputs)
	if observation != nil {
		result.Trust = composerTrustRequirements(
			request.Facts,
			*observation,
			result.Inputs,
		)
	}
	if operation != model.OperationDoctor &&
		observation != nil &&
		!hasBlockingDiagnostic(diagnostics) {
		actions := compatibleActions(
			result,
			*observation,
			providerPlan.Actions,
		)
		policyDiagnostics := actionPolicyDiagnostics(
			result.Policy,
			actions,
			request.Facts,
		)
		result.Diagnostics = append(result.Diagnostics, policyDiagnostics...)
		sortDiagnostics(result.Diagnostics)
		if !hasBlockingDiagnostic(result.Diagnostics) {
			result.Actions = actions
		}
	}

	digest, err := computeDigest(result)
	if err != nil {
		return model.Plan{}, err
	}
	result.Digest = digest

	return result, nil
}

func selectedProviderPlan(
	plans map[string]providers.ProviderPlan,
	name string,
) providers.ProviderPlan {
	if name == "" {
		return providers.ProviderPlan{}
	}
	if selected, found := plans[name]; found {
		return selected
	}
	for candidate, providerPlan := range plans {
		if strings.EqualFold(strings.TrimSpace(candidate), name) {
			return providerPlan
		}
	}

	return providers.ProviderPlan{}
}

func diagnosticsContain(
	diagnostics []model.Diagnostic,
	code string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}

	return false
}

func ambiguousRequirements(
	resolutions []model.RequirementResolution,
	diagnostics []model.Diagnostic,
) ([]model.RequirementResolution, []model.Diagnostic) {
	independentSources := make(map[string]struct{})
	for _, diagnostic := range diagnostics {
		switch diagnostic.Code {
		case "ELEFANTE_REQUIREMENT_CONFLICT",
			"ELEFANTE_REQUIREMENT_INVALID":
			independentSources[sourceKey(diagnostic.Sources)] = struct{}{}
		}
	}
	result := append([]model.RequirementResolution(nil), resolutions...)
	for index := range result {
		key := sourceKey(result[index].Sources)
		if _, independent := independentSources[key]; independent {
			continue
		}
		result[index].Status = model.ResolutionAmbiguous
		result[index].SelectedValue = ""
	}

	filtered := make([]model.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		switch diagnostic.Code {
		case "ELEFANTE_REQUIREMENT_UNAVAILABLE",
			"ELEFANTE_OPTIONAL_EXTENSION_MISSING":
			continue
		default:
			filtered = append(filtered, diagnostic)
		}
	}

	return result, filtered
}

func legacyFrameworkDiagnostics(
	facts model.ProjectFacts,
) []model.Diagnostic {
	laravel := false
	for _, framework := range facts.Frameworks {
		if framework.Kind == model.FrameworkLaravelApplication ||
			framework.Kind == model.FrameworkLaravelPackage {
			laravel = true
			break
		}
	}
	if !laravel {
		return nil
	}

	requirements := append(
		[]model.ComposerLink(nil),
		facts.Composer.Manifest.Requirements...,
	)
	requirements = append(
		requirements,
		facts.Composer.Manifest.DevelopmentRequirements...,
	)
	for _, requirement := range requirements {
		if requirement.Name != "laravel/framework" ||
			!isLegacyLaravelConstraint(requirement.Constraint) {
			continue
		}

		return []model.Diagnostic{
			{
				Code:     "ELEFANTE_LEGACY_FRAMEWORK",
				Severity: model.SeverityWarning,
				Message:  "Laravel 11 is outside the supported execution matrix.",
				Detail:   "The project may execute, but Laravel 11 is maintained as a legacy compatibility case.",
				Hint:     "Use Laravel 12 or Laravel 13 for supported execution.",
				Sources:  []model.SourceReference{requirement.Source},
			},
		}
	}

	return nil
}

func actionPolicyDiagnostics(
	policy model.PlanPolicy,
	actions []model.PlanAction,
	facts model.ProjectFacts,
) []model.Diagnostic {
	var diagnostics []model.Diagnostic
	if policy.Frozen &&
		facts.Composer.Lock.Status == model.ComposerLockMissing {
		diagnostics = append(diagnostics, model.Diagnostic{
			Code:     "ELEFANTE_FROZEN_LOCK_REQUIRED",
			Severity: model.SeverityError,
			Message:  "Frozen synchronization requires an existing Composer lock file.",
			Detail:   "Installing without composer.lock would create project dependency state.",
			Hint:     "Create composer.lock outside frozen mode, then retry.",
			Sources: []model.SourceReference{
				{
					Path: facts.Composer.Manifest.Path,
					Kind: "composer_manifest",
				},
			},
		})
	}
	if policy.Offline {
		for _, action := range actions {
			if action.Network == model.NetworkNone {
				continue
			}

			diagnostics = append(diagnostics, model.Diagnostic{
				Code:     "ELEFANTE_OFFLINE_NETWORK_REQUIRED",
				Severity: model.SeverityError,
				Message:  "The plan requires network access while offline mode is active.",
				Detail:   fmt.Sprintf("Action %q cannot run without a prepared local substitute.", action.Kind),
				Hint:     "Prepare the required artifacts while online, then retry offline.",
				Sources: []model.SourceReference{
					{
						Path: facts.Composer.Manifest.Path,
						Kind: "composer_manifest",
					},
				},
			})
			break
		}
	}

	return diagnostics
}

func selectProvider(
	request Request,
) (model.ProviderSelection, *model.ProviderObservation, []model.Diagnostic) {
	observations := append(
		[]model.ProviderObservation(nil),
		request.Observations...,
	)
	sort.Slice(observations, func(left int, right int) bool {
		return observations[left].Provider < observations[right].Provider
	})

	allowed := normalizedSet(request.Facts.Configuration.Providers.Allowed)
	denied := normalizedSet(request.Facts.Configuration.Providers.Denied)
	candidates := make([]model.ProviderObservation, 0, len(observations))
	for _, observation := range observations {
		name := strings.ToLower(strings.TrimSpace(observation.Provider))
		if !observation.Available {
			continue
		}
		if _, blocked := denied[name]; blocked {
			continue
		}
		if len(allowed) > 0 {
			if _, permitted := allowed[name]; !permitted {
				continue
			}
		}
		observation.Provider = name
		observation = normalizeProviderObservation(
			request.Facts,
			observation,
		)
		candidates = append(candidates, observation)
	}

	if requested := strings.ToLower(strings.TrimSpace(request.Provider)); requested != "" {
		if selected := providerByName(candidates, requested); selected != nil {
			return providerSelection(*selected, "explicit"), selected, nil
		}

		return model.ProviderSelection{}, nil, []model.Diagnostic{
			providerDiagnostic(
				"ELEFANTE_PROVIDER_UNAVAILABLE",
				"The explicitly selected provider is unavailable.",
				requested,
			),
		}
	}

	for _, preferred := range request.Facts.Configuration.Providers.Preferred {
		name := strings.ToLower(strings.TrimSpace(preferred))
		if selected := providerByName(candidates, name); selected != nil {
			return providerSelection(*selected, "configuration"), selected, nil
		}
	}
	for _, marker := range request.Facts.ProviderMarkers {
		name := strings.ToLower(strings.TrimSpace(marker.Provider))
		if selected := providerByName(candidates, name); selected != nil {
			return providerSelection(*selected, "provider_marker"), selected, nil
		}
	}
	if name := strings.ToLower(
		strings.TrimSpace(request.PreviousProvider),
	); name != "" {
		if selected := providerByName(candidates, name); selected != nil {
			return providerSelection(*selected, "workspace_state"), selected, nil
		}
	}
	if name := strings.ToLower(
		strings.TrimSpace(request.DefaultProvider),
	); name != "" {
		if selected := providerByName(candidates, name); selected != nil {
			return providerSelection(*selected, "user_default"), selected, nil
		}
	}

	if len(candidates) > 1 {
		if selected := bestCompatibleProvider(
			request.Facts,
			candidates,
		); selected != nil {
			return providerSelection(*selected, "best_compatible"),
				selected,
				nil
		}
	}

	switch len(candidates) {
	case 0:
		return model.ProviderSelection{}, nil, []model.Diagnostic{
			providerDiagnostic(
				"ELEFANTE_PROVIDER_UNAVAILABLE",
				"No allowed environment provider is available.",
				"",
			),
		}
	case 1:
		return providerSelection(candidates[0], "only_available"),
			&candidates[0],
			nil
	default:
		sources := make([]model.SourceReference, 0, len(candidates))
		for _, candidate := range candidates {
			sources = append(sources, model.SourceReference{
				Path: candidate.Provider,
				Kind: "provider_observation",
			})
		}

		return model.ProviderSelection{}, nil, []model.Diagnostic{
			{
				Code:     "ELEFANTE_PROVIDER_AMBIGUOUS",
				Severity: model.SeverityError,
				Message:  "Several environment providers are equally eligible.",
				Detail:   "Elefante will not select a provider arbitrarily.",
				Hint:     "Pass --provider or configure providers.preferred.",
				Sources:  sources,
			},
		}
	}
}

func normalizeProviderObservation(
	facts model.ProjectFacts,
	observation model.ProviderObservation,
) model.ProviderObservation {
	observation.Capabilities = append(
		[]model.Capability(nil),
		observation.Capabilities...,
	)
	sort.Slice(observation.Capabilities, func(left int, right int) bool {
		return observation.Capabilities[left] <
			observation.Capabilities[right]
	})
	observation.Runtimes = append(
		[]model.RuntimeObservation(nil),
		observation.Runtimes...,
	)
	sort.Slice(observation.Runtimes, func(left int, right int) bool {
		if observation.Runtimes[left].Name !=
			observation.Runtimes[right].Name {
			return observation.Runtimes[left].Name <
				observation.Runtimes[right].Name
		}
		if observation.Runtimes[left].Version !=
			observation.Runtimes[right].Version {
			return versionGreater(
				observation.Runtimes[left].Version,
				observation.Runtimes[right].Version,
			)
		}

		return sourceKey(
			[]model.SourceReference{observation.Runtimes[left].Source},
		) < sourceKey(
			[]model.SourceReference{observation.Runtimes[right].Source},
		)
	})
	observation.Extensions = append(
		[]model.ExtensionObservation(nil),
		observation.Extensions...,
	)
	sort.Slice(observation.Extensions, func(left int, right int) bool {
		if observation.Extensions[left].Name !=
			observation.Extensions[right].Name {
			return observation.Extensions[left].Name <
				observation.Extensions[right].Name
		}
		if observation.Extensions[left].Available !=
			observation.Extensions[right].Available {
			return observation.Extensions[left].Available
		}
		if observation.Extensions[left].Version !=
			observation.Extensions[right].Version {
			return versionGreater(
				observation.Extensions[left].Version,
				observation.Extensions[right].Version,
			)
		}

		return sourceKey(
			[]model.SourceReference{observation.Extensions[left].Source},
		) < sourceKey(
			[]model.SourceReference{observation.Extensions[right].Source},
		)
	})
	observation.Composer = append(
		[]model.ComposerObservation(nil),
		observation.Composer...,
	)
	composerRequirements := composerRequirementGroup(facts)
	sort.Slice(observation.Composer, func(left int, right int) bool {
		leftCompatible := composerObservationCompatible(
			observation.Composer[left],
			composerRequirements,
		)
		rightCompatible := composerObservationCompatible(
			observation.Composer[right],
			composerRequirements,
		)
		if leftCompatible != rightCompatible {
			return leftCompatible
		}
		leftPriority := composerSourcePriority(
			observation.Composer[left].Source,
		)
		rightPriority := composerSourcePriority(
			observation.Composer[right].Source,
		)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if observation.Composer[left].Version !=
			observation.Composer[right].Version {
			return versionGreater(
				observation.Composer[left].Version,
				observation.Composer[right].Version,
			)
		}
		if observation.Composer[left].Identity !=
			observation.Composer[right].Identity {
			return observation.Composer[left].Identity <
				observation.Composer[right].Identity
		}

		return observation.Composer[left].Path <
			observation.Composer[right].Path
	})
	observation.Diagnostics = cloneDiagnostics(observation.Diagnostics)
	sortDiagnostics(observation.Diagnostics)

	return observation
}

func composerRequirementGroup(
	facts model.ProjectFacts,
) []model.Requirement {
	for _, group := range groupRequirements(facts) {
		if group.kind == model.RequirementComposer {
			return group.requirements
		}
	}

	return nil
}

func composerObservationCompatible(
	observation model.ComposerObservation,
	requirements []model.Requirement,
) bool {
	if len(requirements) == 0 {
		return true
	}
	satisfied, err := requirementsSatisfied(
		requirements,
		observation.Version,
	)

	return err == nil && satisfied
}

func composerSourcePriority(source string) int {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "managed", "elefante_managed":
		return 1
	case "system":
		return 3
	default:
		return 2
	}
}

func versionGreater(left string, right string) bool {
	leftVersion, leftErr := constraints.NormalizeVersion(left)
	rightVersion, rightErr := constraints.NormalizeVersion(right)
	if leftErr == nil && rightErr == nil {
		return leftVersion.Compare(rightVersion) > 0
	}

	return left > right
}

func bestCompatibleProvider(
	facts model.ProjectFacts,
	candidates []model.ProviderObservation,
) *model.ProviderObservation {
	bestIndex := -1
	bestScore := -1
	bestReadiness := -1
	tied := false
	for index := range candidates {
		resolutions, diagnostics, err := resolveRequirements(
			facts,
			&candidates[index],
		)
		if err != nil ||
			hasBlockingDiagnostic(diagnostics) ||
			hasBlockingDiagnostic(candidates[index].Diagnostics) {
			continue
		}
		score := 0
		compatible := true
		for _, resolution := range resolutions {
			switch resolution.Status {
			case model.ResolutionSatisfied,
				model.ResolutionLegacy,
				model.ResolutionOptionalMissing:
				score += 2
			case model.ResolutionActionRequired:
				score++
			default:
				compatible = false
			}
		}
		if !compatible {
			continue
		}
		readiness := providerReadinessScore(candidates[index].State)
		switch {
		case score > bestScore ||
			(score == bestScore && readiness > bestReadiness):
			bestIndex = index
			bestScore = score
			bestReadiness = readiness
			tied = false
		case score == bestScore && readiness == bestReadiness:
			tied = true
		}
	}
	if bestIndex < 0 || tied {
		return nil
	}

	return &candidates[bestIndex]
}

func providerReadinessScore(state model.ProviderState) int {
	switch state {
	case "", model.ProviderStateRunning:
		return 2
	case model.ProviderStateStopped:
		return 1
	default:
		return 0
	}
}

func providerByName(
	candidates []model.ProviderObservation,
	name string,
) *model.ProviderObservation {
	for index := range candidates {
		if candidates[index].Provider == name {
			return &candidates[index]
		}
	}

	return nil
}

func providerSelection(
	observation model.ProviderObservation,
	reason string,
) model.ProviderSelection {
	return model.ProviderSelection{
		Name:                   observation.Provider,
		Reason:                 reason,
		ObservationFingerprint: observation.Fingerprint,
	}
}

func providerDiagnostic(
	code string,
	message string,
	provider string,
) model.Diagnostic {
	return model.Diagnostic{
		Code:     code,
		Severity: model.SeverityError,
		Message:  message,
		Hint:     "Inspect provider availability or select another provider.",
		Provider: provider,
	}
}

func optionalRequirementDiagnostic(group requirementGroup) model.Diagnostic {
	return model.Diagnostic{
		Code:     "ELEFANTE_OPTIONAL_EXTENSION_MISSING",
		Severity: model.SeverityWarning,
		Message:  "An optional extension requirement is not satisfied.",
		Detail:   fmt.Sprintf("%s is optional, so planning can continue.", group.name),
		Hint:     "Install the extension to enable the related project capability.",
		Sources:  requirementSources(group.requirements),
	}
}

func resolveRequirements(
	facts model.ProjectFacts,
	observation *model.ProviderObservation,
) ([]model.RequirementResolution, []model.Diagnostic, error) {
	groups := groupRequirements(facts)
	resolutions := make([]model.RequirementResolution, 0, len(groups))
	var diagnostics []model.Diagnostic

	for _, group := range groups {
		selected, available := observedValue(group, observation)
		resolution := model.RequirementResolution{
			Name:       group.name,
			Kind:       group.kind,
			Constraint: joinedConstraints(group.requirements),
			Optional:   group.optional,
			Sources:    requirementSources(group.requirements),
		}
		if err := validateRequirementGroup(group); err != nil {
			resolution.Status = model.ResolutionBlocked
			diagnostics = append(
				diagnostics,
				invalidRequirementDiagnostic(group, err),
			)
			resolutions = append(resolutions, resolution)
			continue
		}
		conflicting, err := requirementsConflict(group.requirements)
		if err != nil {
			return nil, nil, err
		}
		if conflicting {
			resolution.Status = model.ResolutionBlocked
			diagnostics = append(
				diagnostics,
				requirementConflictDiagnostic(group),
			)
			resolutions = append(resolutions, resolution)
			continue
		}
		switch {
		case !available && group.optional:
			resolution.Status = model.ResolutionOptionalMissing
			diagnostics = append(
				diagnostics,
				optionalRequirementDiagnostic(group),
			)
		case !available:
			target, actionable, err := preparationTarget(group, observation)
			if err != nil {
				return nil, nil, err
			}
			if actionable {
				resolution.Status = model.ResolutionActionRequired
				resolution.SelectedValue = target
			} else {
				resolution.Status = model.ResolutionBlocked
				diagnostics = append(
					diagnostics,
					requirementDiagnostic(
						"ELEFANTE_REQUIREMENT_UNAVAILABLE",
						group,
						"",
					),
				)
			}
		default:
			satisfied, err := requirementsSatisfied(group.requirements, selected)
			if err != nil {
				return nil, nil, err
			}
			resolution.SelectedValue = selected
			if satisfied {
				legacy, err := legacyRuntime(group, selected)
				if err != nil {
					return nil, nil, err
				}
				if legacy {
					resolution.Status = model.ResolutionLegacy
					diagnostics = append(
						diagnostics,
						legacyRuntimeDiagnostic(group, selected),
					)
				} else {
					resolution.Status = model.ResolutionSatisfied
				}
			} else if group.optional {
				resolution.Status = model.ResolutionOptionalMissing
				diagnostics = append(
					diagnostics,
					optionalRequirementDiagnostic(group),
				)
			} else {
				target, actionable, err := preparationTarget(group, observation)
				if err != nil {
					return nil, nil, err
				}
				if actionable {
					resolution.Status = model.ResolutionActionRequired
					resolution.SelectedValue = target
				} else {
					resolution.Status = model.ResolutionBlocked
					diagnostics = append(
						diagnostics,
						requirementDiagnostic(
							"ELEFANTE_REQUIREMENT_INCOMPATIBLE",
							group,
							selected,
						),
					)
				}
			}
		}
		resolutions = append(resolutions, resolution)
	}

	return resolutions, diagnostics, nil
}

func validateRequirementGroup(group requirementGroup) error {
	for _, requirement := range group.requirements {
		if _, err := constraints.Parse(requirement.Constraint); err != nil {
			return err
		}
	}

	return nil
}

func preparationTarget(
	group requirementGroup,
	observation *model.ProviderObservation,
) (string, bool, error) {
	if observation == nil {
		return "", false, nil
	}

	switch group.kind {
	case model.RequirementPHP:
		if !hasCapability(*observation, model.CapabilityInstallRuntime) {
			return "", false, nil
		}

		return supportedRuntimeTarget(group.requirements)
	case model.RequirementExtension:
		if !hasCapability(*observation, model.CapabilityInstallExtension) {
			return "", false, nil
		}

		return joinedConstraints(group.requirements), true, nil
	default:
		return "", false, nil
	}
}

func hasCapability(
	observation model.ProviderObservation,
	expected model.Capability,
) bool {
	for _, capability := range observation.Capabilities {
		if capability == expected {
			return true
		}
	}

	return false
}

func requirementsConflict(
	requirements []model.Requirement,
) (bool, error) {
	var positive []model.Requirement
	for _, requirement := range requirements {
		if !requirementIsConflict(requirement.Scope) {
			positive = append(positive, requirement)
		}
	}
	for left := 0; left < len(positive); left++ {
		for right := left + 1; right < len(positive); right++ {
			intersects, err := constraints.Intersects(
				positive[left].Constraint,
				positive[right].Constraint,
			)
			if err != nil {
				return false, err
			}
			if !intersects {
				return true, nil
			}
		}
	}

	return false, nil
}

func groupRequirements(facts model.ProjectFacts) []requirementGroup {
	grouped := make(map[string]*requirementGroup)
	for _, requirement := range facts.Composer.PlatformRequirements {
		key := string(requirement.Kind) + "\x00" + requirement.Name
		group := grouped[key]
		if group == nil {
			group = &requirementGroup{
				name:     requirement.Name,
				kind:     requirement.Kind,
				optional: requirement.Optional,
			}
			grouped[key] = group
		} else {
			group.optional = group.optional && requirement.Optional
		}
		group.requirements = append(group.requirements, requirement)
	}

	for _, version := range facts.VersionFiles {
		if version.Runtime != "php" {
			continue
		}
		key := string(model.RequirementPHP) + "\x00php"
		group := grouped[key]
		if group == nil {
			group = &requirementGroup{
				name: "php",
				kind: model.RequirementPHP,
			}
			grouped[key] = group
		}
		group.requirements = append(group.requirements, model.Requirement{
			Name:       "php",
			Kind:       model.RequirementPHP,
			Constraint: version.Version,
			Scope:      model.RequirementScopeVersionFile,
			Sources:    []model.SourceReference{version.Source},
		})
	}

	if constraint := strings.TrimSpace(
		facts.Configuration.Composer.Constraint,
	); constraint != "" {
		key := string(model.RequirementComposer) + "\x00composer"
		group := grouped[key]
		if group == nil {
			group = &requirementGroup{
				name: "composer",
				kind: model.RequirementComposer,
			}
			grouped[key] = group
		}
		group.requirements = append(group.requirements, model.Requirement{
			Name:       "composer",
			Kind:       model.RequirementComposer,
			Constraint: constraint,
			Scope:      model.RequirementScopeConfiguration,
			Sources: []model.SourceReference{
				{
					Path:  facts.Configuration.Path,
					Kind:  "elefante_config",
					Field: "/composer/constraint",
				},
			},
		})
	}

	for _, extension := range facts.Configuration.Extensions.Optional {
		key := string(model.RequirementExtension) + "\x00" + extension
		group := grouped[key]
		if group == nil {
			group = &requirementGroup{
				name:     extension,
				kind:     model.RequirementExtension,
				optional: true,
			}
			grouped[key] = group
		}
		group.requirements = append(group.requirements, model.Requirement{
			Name:       extension,
			Kind:       model.RequirementExtension,
			Constraint: "*",
			Scope:      model.RequirementScopeConfiguration,
			Optional:   true,
			Sources: []model.SourceReference{
				{
					Path:  facts.Configuration.Path,
					Kind:  "elefante_config",
					Field: "/extensions/optional",
				},
			},
		})
	}

	result := make([]requirementGroup, 0, len(grouped))
	for _, group := range grouped {
		sort.Slice(group.requirements, func(left int, right int) bool {
			leftRequirement := group.requirements[left]
			rightRequirement := group.requirements[right]
			if leftRequirement.Scope != rightRequirement.Scope {
				return leftRequirement.Scope < rightRequirement.Scope
			}
			if leftRequirement.Constraint != rightRequirement.Constraint {
				return leftRequirement.Constraint < rightRequirement.Constraint
			}

			return leftRequirement.Package < rightRequirement.Package
		})
		result = append(result, *group)
	}
	sort.Slice(result, func(left int, right int) bool {
		if result[left].kind != result[right].kind {
			return result[left].kind < result[right].kind
		}

		return result[left].name < result[right].name
	})

	return result
}

func observedValue(
	group requirementGroup,
	observation *model.ProviderObservation,
) (string, bool) {
	if observation == nil {
		return "", false
	}

	switch group.kind {
	case model.RequirementPHP, model.RequirementPHPSubtype:
		for _, runtime := range observation.Runtimes {
			if runtime.Name == group.name ||
				(group.kind == model.RequirementPHP && runtime.Name == "php") {
				return runtime.Version, runtime.Version != ""
			}
		}
	case model.RequirementExtension:
		for _, extension := range observation.Extensions {
			if extension.Name != group.name || !extension.Available {
				continue
			}
			if extension.Version == "" &&
				joinedConstraints(group.requirements) == "*" {
				return "0", true
			}

			return extension.Version, extension.Version != ""
		}
	case model.RequirementComposer:
		if len(observation.Composer) > 0 {
			return observation.Composer[0].Version,
				observation.Composer[0].Version != ""
		}
	case model.RequirementComposerPluginAPI:
		if len(observation.Composer) > 0 {
			return observation.Composer[0].PluginAPI,
				observation.Composer[0].PluginAPI != ""
		}
	case model.RequirementComposerRuntimeAPI:
		if len(observation.Composer) > 0 {
			return observation.Composer[0].RuntimeAPI,
				observation.Composer[0].RuntimeAPI != ""
		}
	}

	return "", false
}

func requirementsSatisfied(
	requirements []model.Requirement,
	selected string,
) (bool, error) {
	for _, requirement := range requirements {
		matches, err := constraints.EvaluateRequirement(requirement, selected)
		if err != nil {
			return false, err
		}
		if requirementIsConflict(requirement.Scope) {
			matches = !matches
		}
		if !matches {
			return false, nil
		}
	}

	return true, nil
}

func requirementIsConflict(scope model.RequirementScope) bool {
	switch scope {
	case model.RequirementScopeRootConflict,
		model.RequirementScopeLockedPackageConflict,
		model.RequirementScopeLockedDevelopmentPackageConflict:
		return true
	default:
		return false
	}
}

func requirementDiagnostic(
	code string,
	group requirementGroup,
	selected string,
) model.Diagnostic {
	detail := "No compatible observed value is available."
	if selected != "" {
		detail = fmt.Sprintf(
			"Observed %s does not satisfy %s.",
			selected,
			joinedConstraints(group.requirements),
		)
	}

	return model.Diagnostic{
		Code:     code,
		Severity: model.SeverityError,
		Message:  fmt.Sprintf("Requirement %q is not satisfied.", group.name),
		Detail:   detail,
		Hint:     "Select a compatible provider or prepare the required platform value.",
		Sources:  requirementSources(group.requirements),
	}
}

func invalidRequirementDiagnostic(
	group requirementGroup,
	err error,
) model.Diagnostic {
	return model.Diagnostic{
		Code:     "ELEFANTE_REQUIREMENT_INVALID",
		Severity: model.SeverityError,
		Message:  fmt.Sprintf("Requirement %q has an invalid constraint.", group.name),
		Detail:   err.Error(),
		Hint:     "Correct the constraint at its reported source.",
		Sources:  requirementSources(group.requirements),
	}
}

func legacyRuntimeDiagnostic(
	group requirementGroup,
	selected string,
) model.Diagnostic {
	return model.Diagnostic{
		Code:     "ELEFANTE_LEGACY_RUNTIME",
		Severity: model.SeverityWarning,
		Message:  fmt.Sprintf("Runtime %s is outside the supported matrix.", selected),
		Detail:   "The project may execute, but this runtime is maintained as a legacy compatibility case.",
		Hint:     "Use PHP 8.3 through PHP 8.5 for supported execution.",
		Sources:  requirementSources(group.requirements),
	}
}

func requirementConflictDiagnostic(group requirementGroup) model.Diagnostic {
	return model.Diagnostic{
		Code:     "ELEFANTE_REQUIREMENT_CONFLICT",
		Severity: model.SeverityError,
		Message:  fmt.Sprintf("Requirement %q has conflicting intent.", group.name),
		Detail: fmt.Sprintf(
			"No value can satisfy %s.",
			joinedConstraints(group.requirements),
		),
		Hint:    "Align Composer metadata, version files, and Elefante policy.",
		Sources: requirementSources(group.requirements),
	}
}

func joinedConstraints(requirements []model.Requirement) string {
	values := make([]string, 0, len(requirements))
	seen := make(map[string]struct{}, len(requirements))
	for _, requirement := range requirements {
		value := requirement.Constraint
		if requirementIsConflict(requirement.Scope) {
			value = "conflict " + value
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	sort.Strings(values)

	return strings.Join(values, " && ")
}

func requirementSources(
	requirements []model.Requirement,
) []model.SourceReference {
	var sources []model.SourceReference
	for _, requirement := range requirements {
		sources = append(sources, requirement.Sources...)
	}
	sortSources(sources)

	return deduplicateSources(sources)
}

func compatibleActions(
	result model.Plan,
	observation model.ProviderObservation,
	providerActions []model.PlanAction,
) []model.PlanAction {
	composerIdentity := ""
	if len(observation.Composer) > 0 {
		composerIdentity = observation.Composer[0].Identity
	}
	composerTrust := composerActionTrust(result.Trust)
	actions := clonePlanActions(providerActions)
	for _, resolution := range result.Requirements {
		if resolution.Status != model.ResolutionActionRequired {
			continue
		}
		switch resolution.Kind {
		case model.RequirementPHP:
			actions = append(actions, model.PlanAction{
				Kind:    model.ActionPrepareRuntime,
				Summary: "Prepare a compatible PHP runtime.",
				Effect:  model.EffectProviderMutation,
				Network: model.NetworkRequired,
				Trust:   model.TrustNone,
				Inputs: []model.ActionInput{
					{Name: "provider", Value: result.Provider.Name},
					{Name: "runtime", Value: resolution.Name},
					{Name: "version", Value: resolution.SelectedValue},
				},
				ExpectedOutputs: []model.ActionOutput{
					{Name: "runtime", Value: resolution.SelectedValue},
				},
			})
		case model.RequirementExtension:
			actions = append(actions, model.PlanAction{
				Kind:    model.ActionPrepareExtension,
				Summary: "Prepare a required PHP extension.",
				Effect:  model.EffectProviderMutation,
				Network: model.NetworkRequired,
				Trust:   model.TrustNone,
				Inputs: []model.ActionInput{
					{Name: "extension", Value: resolution.Name},
					{Name: "provider", Value: result.Provider.Name},
					{Name: "version", Value: resolution.SelectedValue},
				},
				ExpectedOutputs: []model.ActionOutput{
					{Name: "extension", Value: resolution.Name},
				},
			})
		}
	}
	actions = append(actions, []model.PlanAction{
		{
			Kind:    model.ActionInstallDependencies,
			Summary: "Install Composer dependencies.",
			Effect:  model.EffectProjectCodeExecution,
			Network: model.NetworkRequired,
			Trust:   composerTrust,
			Inputs: []model.ActionInput{
				{Name: "composer", Value: composerIdentity},
				{Name: "provider", Value: result.Provider.Name},
				{Name: "working_directory", Value: result.Project.ComposerRoot},
			},
			ExpectedOutputs: []model.ActionOutput{
				{Name: "dependencies", Value: "installed"},
			},
		},
		{
			Kind:    model.ActionVerifyPlatform,
			Summary: "Verify the installed Composer platform requirements.",
			Effect:  model.EffectRead,
			Network: model.NetworkNone,
			Trust:   model.TrustNone,
			Inputs: []model.ActionInput{
				{Name: "provider", Value: result.Provider.Name},
				{Name: "working_directory", Value: result.Project.ComposerRoot},
			},
			ExpectedOutputs: []model.ActionOutput{
				{Name: "platform", Value: "verified"},
			},
		},
		{
			Kind:       model.ActionRecordState,
			Summary:    "Record the successful local environment.",
			Effect:     model.EffectLocalStateMutation,
			Network:    model.NetworkNone,
			Trust:      model.TrustNone,
			Reversible: true,
			Inputs: []model.ActionInput{
				{Name: "project", Value: result.Project.IdentityKey},
				{Name: "provider", Value: result.Provider.Name},
			},
			ExpectedOutputs: []model.ActionOutput{
				{Name: "workspace_state", Value: "recorded"},
			},
		},
	}...)

	for index := range actions {
		actions[index].ID = ""
		actions[index].Dependencies = nil
		sortActionInputs(actions[index].Inputs)
		sortActionOutputs(actions[index].ExpectedOutputs)
	}
	sort.Slice(actions, func(left int, right int) bool {
		leftPhase := actionPhase(actions[left].Kind)
		rightPhase := actionPhase(actions[right].Kind)
		if leftPhase != rightPhase {
			return leftPhase < rightPhase
		}
		if actions[left].Kind != actions[right].Kind {
			return actions[left].Kind < actions[right].Kind
		}

		return actionInputKey(actions[left].Inputs) <
			actionInputKey(actions[right].Inputs)
	})
	for index := range actions {
		actions[index].ID = actionID(index, actions[index])
		if index > 0 {
			actions[index].Dependencies = []string{actions[index-1].ID}
		}
	}

	return actions
}

func clonePlanActions(actions []model.PlanAction) []model.PlanAction {
	result := make([]model.PlanAction, len(actions))
	for index, action := range actions {
		result[index] = action
		result[index].Inputs = append(
			[]model.ActionInput(nil),
			action.Inputs...,
		)
		result[index].ExpectedOutputs = append(
			[]model.ActionOutput(nil),
			action.ExpectedOutputs...,
		)
		result[index].Dependencies = append(
			[]string(nil),
			action.Dependencies...,
		)
	}

	return result
}

func composerTrustRequirements(
	facts model.ProjectFacts,
	observation model.ProviderObservation,
	inputs []model.InputFingerprint,
) []model.TrustRequirement {
	if len(facts.Composer.Plugins) == 0 &&
		len(facts.Composer.Scripts) == 0 {
		return nil
	}

	type pluginIdentity struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Development bool   `json:"development"`
	}
	type scriptIdentity struct {
		Name          string `json:"name"`
		CommandCount  int    `json:"command_count"`
		ContentSHA256 string `json:"content_sha256"`
	}
	canonical := struct {
		Schema           string                   `json:"schema"`
		Inputs           []model.InputFingerprint `json:"inputs"`
		ComposerIdentity string                   `json:"composer_identity"`
		Plugins          []pluginIdentity         `json:"plugins"`
		Scripts          []scriptIdentity         `json:"scripts"`
	}{
		Schema: "elefante.composer-trust/v1",
		Inputs: cloneFingerprints(
			inputs,
		),
	}
	if len(observation.Composer) > 0 {
		canonical.ComposerIdentity = observation.Composer[0].Identity
	}
	for _, plugin := range facts.Composer.Plugins {
		canonical.Plugins = append(canonical.Plugins, pluginIdentity{
			Name:        plugin.Name,
			Version:     plugin.Version,
			Development: plugin.Development,
		})
	}
	sort.Slice(canonical.Plugins, func(left int, right int) bool {
		if canonical.Plugins[left].Name != canonical.Plugins[right].Name {
			return canonical.Plugins[left].Name < canonical.Plugins[right].Name
		}
		if canonical.Plugins[left].Version != canonical.Plugins[right].Version {
			return canonical.Plugins[left].Version <
				canonical.Plugins[right].Version
		}

		return !canonical.Plugins[left].Development &&
			canonical.Plugins[right].Development
	})
	for _, script := range facts.Composer.Scripts {
		canonical.Scripts = append(canonical.Scripts, scriptIdentity{
			Name:          script.Name,
			CommandCount:  script.CommandCount,
			ContentSHA256: script.ContentSHA256,
		})
	}
	sort.Slice(canonical.Scripts, func(left int, right int) bool {
		if canonical.Scripts[left].Name != canonical.Scripts[right].Name {
			return canonical.Scripts[left].Name <
				canonical.Scripts[right].Name
		}
		if canonical.Scripts[left].ContentSHA256 !=
			canonical.Scripts[right].ContentSHA256 {
			return canonical.Scripts[left].ContentSHA256 <
				canonical.Scripts[right].ContentSHA256
		}

		return canonical.Scripts[left].CommandCount <
			canonical.Scripts[right].CommandCount
	})
	encoded, _ := json.Marshal(canonical)
	sum := sha256.Sum256(encoded)
	fingerprint := "sha256:" + hex.EncodeToString(sum[:])

	var requirements []model.TrustRequirement
	if len(facts.Composer.Plugins) > 0 {
		var sources []model.SourceReference
		for _, plugin := range facts.Composer.Plugins {
			sources = append(sources, plugin.Source)
		}
		sortSources(sources)
		requirements = append(requirements, model.TrustRequirement{
			Class:       model.TrustComposerPlugins,
			Fingerprint: fingerprint,
			Sources:     deduplicateSources(sources),
		})
	}
	if len(facts.Composer.Scripts) > 0 {
		var sources []model.SourceReference
		for _, script := range facts.Composer.Scripts {
			sources = append(sources, script.Source)
		}
		sortSources(sources)
		requirements = append(requirements, model.TrustRequirement{
			Class:       model.TrustComposerScripts,
			Fingerprint: fingerprint,
			Sources:     deduplicateSources(sources),
		})
	}

	return requirements
}

func composerActionTrust(
	requirements []model.TrustRequirement,
) model.TrustClass {
	for _, requirement := range requirements {
		if requirement.Class == model.TrustComposerPlugins {
			return model.TrustComposerPlugins
		}
	}
	for _, requirement := range requirements {
		if requirement.Class == model.TrustComposerScripts {
			return model.TrustComposerScripts
		}
	}

	return model.TrustNone
}

func actionPhase(kind model.ActionKind) int {
	switch kind {
	case model.ActionPrepareCache:
		return 1
	case model.ActionPrepareProvider:
		return 2
	case model.ActionPrepareRuntime:
		return 3
	case model.ActionPrepareExtension:
		return 4
	case model.ActionPrepareComposer:
		return 5
	case model.ActionInstallDependencies:
		return 6
	case model.ActionVerifyPlatform:
		return 7
	case model.ActionRecordState:
		return 8
	default:
		return 9
	}
}

func actionInputKey(inputs []model.ActionInput) string {
	encoded, _ := json.Marshal(inputs)

	return string(encoded)
}

func actionID(index int, action model.PlanAction) string {
	input := struct {
		Index  int                 `json:"index"`
		Kind   model.ActionKind    `json:"kind"`
		Inputs []model.ActionInput `json:"inputs"`
	}{
		Index:  index,
		Kind:   action.Kind,
		Inputs: action.Inputs,
	}
	encoded, _ := json.Marshal(input)
	sum := sha256.Sum256(encoded)

	return fmt.Sprintf(
		"%02d-%s-%s",
		index+1,
		action.Kind,
		hex.EncodeToString(sum[:6]),
	)
}

func computeDigest(plan model.Plan) (string, error) {
	canonical := struct {
		ProtocolVersion string     `json:"protocol_version"`
		MatrixVersion   string     `json:"matrix_version"`
		Plan            model.Plan `json:"plan"`
	}{
		ProtocolVersion: model.EventSchema,
		MatrixVersion:   supportMatrixVersion,
		Plan:            clonePlanForDigest(plan),
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode canonical plan: %w", err)
	}
	sum := sha256.Sum256(encoded)

	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func clonePlanForDigest(plan model.Plan) model.Plan {
	canonical := plan
	canonical.Digest = ""
	canonical.Actions = append([]model.PlanAction(nil), plan.Actions...)
	for index := range canonical.Actions {
		canonical.Actions[index].Summary = ""
		canonical.Actions[index].Inputs = append(
			[]model.ActionInput(nil),
			canonical.Actions[index].Inputs...,
		)
		canonical.Actions[index].ExpectedOutputs = append(
			[]model.ActionOutput(nil),
			canonical.Actions[index].ExpectedOutputs...,
		)
		canonical.Actions[index].Dependencies = append(
			[]string(nil),
			canonical.Actions[index].Dependencies...,
		)
	}
	canonical.Diagnostics = cloneDiagnostics(plan.Diagnostics)
	for index := range canonical.Diagnostics {
		canonical.Diagnostics[index].Message = ""
		canonical.Diagnostics[index].Detail = ""
		canonical.Diagnostics[index].Hint = ""
	}
	canonical.Requirements = append(
		[]model.RequirementResolution(nil),
		plan.Requirements...,
	)
	for index := range canonical.Requirements {
		canonical.Requirements[index].Sources = append(
			[]model.SourceReference(nil),
			canonical.Requirements[index].Sources...,
		)
	}
	canonical.Inputs = cloneFingerprints(plan.Inputs)
	canonical.Trust = append(
		[]model.TrustRequirement(nil),
		plan.Trust...,
	)
	for index := range canonical.Trust {
		canonical.Trust[index].Sources = append(
			[]model.SourceReference(nil),
			canonical.Trust[index].Sources...,
		)
	}

	return canonical
}

func hasBlockingDiagnostic(diagnostics []model.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == model.SeverityError {
			return true
		}
	}

	return false
}

func normalizedSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[strings.ToLower(strings.TrimSpace(value))] = struct{}{}
	}

	return result
}

func cloneDiagnostics(values []model.Diagnostic) []model.Diagnostic {
	result := append([]model.Diagnostic(nil), values...)
	for index := range result {
		result[index].Sources = append(
			[]model.SourceReference(nil),
			result[index].Sources...,
		)
		sortSources(result[index].Sources)
	}

	return result
}

func cloneFingerprints(
	values []model.InputFingerprint,
) []model.InputFingerprint {
	return append([]model.InputFingerprint(nil), values...)
}

func sortDiagnostics(values []model.Diagnostic) {
	for index := range values {
		sortSources(values[index].Sources)
	}
	sort.Slice(values, func(left int, right int) bool {
		if values[left].Severity != values[right].Severity {
			return values[left].Severity < values[right].Severity
		}
		if values[left].Code != values[right].Code {
			return values[left].Code < values[right].Code
		}
		if values[left].Provider != values[right].Provider {
			return values[left].Provider < values[right].Provider
		}
		if values[left].Retryable != values[right].Retryable {
			return !values[left].Retryable && values[right].Retryable
		}

		return sourceKey(values[left].Sources) <
			sourceKey(values[right].Sources)
	})
}

func sourceKey(sources []model.SourceReference) string {
	encoded, _ := json.Marshal(sources)

	return string(encoded)
}

func sortFingerprints(values []model.InputFingerprint) {
	sort.Slice(values, func(left int, right int) bool {
		if values[left].Kind != values[right].Kind {
			return values[left].Kind < values[right].Kind
		}

		return values[left].Path < values[right].Path
	})
}

func sortSources(values []model.SourceReference) {
	sort.Slice(values, func(left int, right int) bool {
		if values[left].Path != values[right].Path {
			return values[left].Path < values[right].Path
		}
		if values[left].Kind != values[right].Kind {
			return values[left].Kind < values[right].Kind
		}
		if values[left].Field != values[right].Field {
			return values[left].Field < values[right].Field
		}

		return values[left].Line < values[right].Line
	})
}

func deduplicateSources(values []model.SourceReference) []model.SourceReference {
	result := make([]model.SourceReference, 0, len(values))
	for _, value := range values {
		if len(result) > 0 && result[len(result)-1] == value {
			continue
		}
		result = append(result, value)
	}

	return result
}

func sortActionInputs(values []model.ActionInput) {
	sort.Slice(values, func(left int, right int) bool {
		if values[left].Name != values[right].Name {
			return values[left].Name < values[right].Name
		}

		return values[left].Value < values[right].Value
	})
}

func sortActionOutputs(values []model.ActionOutput) {
	sort.Slice(values, func(left int, right int) bool {
		if values[left].Name != values[right].Name {
			return values[left].Name < values[right].Name
		}

		return values[left].Value < values[right].Value
	})
}
