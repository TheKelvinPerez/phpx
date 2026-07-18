package app

import (
	"context"
	"strings"

	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/executor"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
)

type SyncActionServiceDependencies struct {
	Providers       []providers.Provider
	Runner          executor.Runner
	AcquireComposer AcquireComposer
}

type SyncActionService struct {
	providers       []providers.Provider
	runner          executor.Runner
	acquireComposer AcquireComposer
}

type AcquireComposer func(
	context.Context,
	composer.AcquireRequest,
) (composer.Executable, error)

func NewSyncActionService(
	dependencies SyncActionServiceDependencies,
) *SyncActionService {
	runner := dependencies.Runner
	if runner == nil {
		runner = executor.OSRunner{}
	}

	return &SyncActionService{
		providers: append(
			[]providers.Provider(nil),
			dependencies.Providers...,
		),
		runner:          runner,
		acquireComposer: dependencies.AcquireComposer,
	}
}

func (service *SyncActionService) Execute(
	ctx context.Context,
	execution SyncActionExecution,
) (SyncActionResult, error) {
	if service == nil || service.runner == nil {
		return SyncActionResult{}, model.NewError(
			model.ErrorInternal,
			"The synchronization action service is not configured.",
		)
	}
	provider, err := service.selectedProvider(execution.Analysis.Plan)
	if err != nil {
		return SyncActionResult{}, err
	}
	if err := validateActionTrust(
		execution.Action,
		execution.Analysis.Plan.Trust,
	); err != nil {
		return SyncActionResult{}, err
	}

	switch execution.Action.Kind {
	case model.ActionPrepareComposer:
		return service.prepareComposer(ctx, execution)
	case model.ActionInstallDependencies, model.ActionVerifyPlatform:
		return service.executeComposer(ctx, provider, execution)
	default:
		result, err := provider.Apply(
			ctx,
			providers.ProviderAction{Action: execution.Action},
			providers.ActionRuntime{},
		)
		if err != nil {
			return SyncActionResult{}, err
		}

		actionResult := SyncActionResult{
			Outputs: append(
				[]model.ActionOutput(nil),
				result.Outputs...,
			),
		}
		if result.Compensation != nil {
			actionResult.Compensation = &SyncCompensation{
				Safe:   result.Compensation.Safe,
				Action: result.Compensation.Action,
			}
		}

		return actionResult, nil
	}
}

func (service *SyncActionService) prepareComposer(
	ctx context.Context,
	execution SyncActionExecution,
) (SyncActionResult, error) {
	release, err := managedRelease(execution.Action)
	if err != nil {
		return SyncActionResult{}, err
	}
	executable, err := service.acquireManagedComposer(
		ctx,
		composer.AcquireRequest{
			Release: release,
			Offline: execution.Analysis.Plan.Policy.Offline,
		},
	)
	if err != nil {
		return SyncActionResult{}, err
	}
	if err := validateManagedExecutable(executable, release); err != nil {
		return SyncActionResult{}, err
	}

	return SyncActionResult{
		Outputs: append(
			[]model.ActionOutput(nil),
			execution.Action.ExpectedOutputs...,
		),
	}, nil
}

func (service *SyncActionService) Compensate(
	ctx context.Context,
	execution SyncCompensationExecution,
) error {
	if !execution.Compensation.Safe {
		return model.NewError(
			model.ErrorSync,
			"The provider did not prove this compensation action safe.",
		)
	}
	provider, err := service.selectedProvider(execution.Analysis.Plan)
	if err != nil {
		return err
	}
	_, err = provider.Apply(
		ctx,
		providers.ProviderAction{
			Action: execution.Compensation.Action,
		},
		providers.ActionRuntime{},
	)

	return err
}

func (service *SyncActionService) executeComposer(
	ctx context.Context,
	provider providers.Provider,
	execution SyncActionExecution,
) (SyncActionResult, error) {
	composerExecutable, err := selectedComposer(
		execution.Analysis,
		execution.Action,
	)
	if err != nil {
		return SyncActionResult{}, err
	}
	if strings.EqualFold(
		composerExecutable.Source,
		composer.SourceManaged,
	) {
		release, err := plannedManagedRelease(execution.Analysis.Plan)
		if err != nil {
			return SyncActionResult{}, err
		}
		verified, err := service.acquireManagedComposer(
			ctx,
			composer.AcquireRequest{
				Release: release,
				Offline: true,
			},
		)
		if err != nil {
			return SyncActionResult{}, err
		}
		if err := validateManagedExecutable(verified, release); err != nil {
			return SyncActionResult{}, err
		}
		if verified.Identity != composerExecutable.Identity {
			return SyncActionResult{}, model.NewError(
				model.ErrorArtifact,
				"The verified managed Composer identity differs from the approved plan.",
			)
		}
		composerExecutable.Path = verified.Path
	}
	arguments := []string{"install"}
	if execution.Action.Kind == model.ActionVerifyPlatform {
		arguments = []string{"check-platform-reqs"}
	} else if execution.NonInteractive {
		arguments = append(arguments, "--no-interaction")
	}
	workingDirectory := actionInput(
		execution.Action,
		"working_directory",
	)
	if strings.TrimSpace(workingDirectory) == "" {
		return SyncActionResult{}, model.NewError(
			model.ErrorSync,
			"The Composer action has no working directory.",
		)
	}
	specification, err := provider.ExecutionSpec(
		ctx,
		providers.ExecutionRequest{
			Executable:       composerExecutable.Path,
			Arguments:        arguments,
			WorkingDirectory: workingDirectory,
		},
	)
	if err != nil {
		return SyncActionResult{}, err
	}
	if strings.TrimSpace(specification.Executable) == "" {
		return SyncActionResult{}, model.NewError(
			model.ErrorProvider,
			"The selected provider returned an empty Composer execution specification.",
		)
	}
	if _, err := service.runner.Output(ctx, executor.Command{
		Executable: specification.Executable,
		Arguments: append(
			[]string(nil),
			specification.Arguments...,
		),
		WorkingDirectory: specification.WorkingDirectory,
		Environment: append(
			[]string(nil),
			specification.Environment...,
		),
	}); err != nil {
		commandError := model.WrapError(
			model.ErrorSync,
			"Official Composer execution failed.",
			err,
		)
		commandError.Provider = execution.Analysis.Plan.Provider.Name

		return SyncActionResult{}, commandError
	}

	return SyncActionResult{
		Outputs: append(
			[]model.ActionOutput(nil),
			execution.Action.ExpectedOutputs...,
		),
	}, nil
}

func (service *SyncActionService) acquireManagedComposer(
	ctx context.Context,
	request composer.AcquireRequest,
) (composer.Executable, error) {
	if service.acquireComposer == nil {
		return composer.Executable{}, model.NewError(
			model.ErrorArtifact,
			"Managed Composer acquisition is not configured.",
		)
	}

	return service.acquireComposer(ctx, request)
}

func plannedManagedRelease(plan model.Plan) (composer.Release, error) {
	for _, action := range plan.Actions {
		if action.Kind == model.ActionPrepareComposer {
			return managedRelease(action)
		}
	}

	return composer.Release{}, model.NewError(
		model.ErrorArtifact,
		"The approved plan has no managed Composer preparation action.",
	)
}

func managedRelease(action model.PlanAction) (composer.Release, error) {
	release := composer.Release{
		Version:     actionInput(action, "version"),
		URL:         actionInput(action, "url"),
		SHA256:      actionInput(action, "sha256"),
		MetadataURL: actionInput(action, "metadata_url"),
	}
	if strings.TrimSpace(release.Version) == "" ||
		strings.TrimSpace(release.URL) == "" ||
		strings.TrimSpace(release.SHA256) == "" ||
		strings.TrimSpace(release.MetadataURL) == "" {
		return composer.Release{}, model.NewError(
			model.ErrorArtifact,
			"The managed Composer action does not identify an exact official release.",
		)
	}

	return release, nil
}

func validateManagedExecutable(
	executable composer.Executable,
	release composer.Release,
) error {
	if executable.Source != composer.SourceManaged ||
		executable.Identity != release.SHA256 ||
		executable.SHA256 != release.SHA256 ||
		strings.TrimSpace(executable.Path) == "" {
		return model.NewError(
			model.ErrorArtifact,
			"Managed Composer acquisition did not return the approved verified executable.",
		)
	}

	return nil
}

func (service *SyncActionService) selectedProvider(
	plan model.Plan,
) (providers.Provider, error) {
	for _, candidate := range service.providers {
		if strings.EqualFold(
			strings.TrimSpace(candidate.Name()),
			strings.TrimSpace(plan.Provider.Name),
		) {
			return candidate, nil
		}
	}
	commandError := model.NewError(
		model.ErrorProvider,
		"The approved synchronization provider is not registered.",
	)
	commandError.Provider = plan.Provider.Name

	return nil, commandError
}

func selectedComposer(
	analysis Analysis,
	action model.PlanAction,
) (model.ComposerObservation, error) {
	identity := actionInput(action, "composer")
	if identity == "" {
		for _, plannedAction := range analysis.Plan.Actions {
			if plannedAction.Kind == model.ActionInstallDependencies {
				identity = actionInput(plannedAction, "composer")
				break
			}
		}
	}
	for _, observation := range analysis.Observations {
		if !strings.EqualFold(
			observation.Provider,
			analysis.Plan.Provider.Name,
		) {
			continue
		}
		for _, candidate := range observation.Composer {
			if candidate.Identity == identity &&
				strings.TrimSpace(candidate.Path) != "" {
				return candidate, nil
			}
		}
	}

	return model.ComposerObservation{}, model.NewError(
		model.ErrorArtifact,
		"The approved Composer executable is not available from the selected provider.",
	)
}

func actionInput(action model.PlanAction, name string) string {
	for _, input := range action.Inputs {
		if input.Name == name {
			return input.Value
		}
	}

	return ""
}

func validateActionTrust(
	action model.PlanAction,
	requirements []model.TrustRequirement,
) error {
	if action.Trust == model.TrustNone || action.Trust == "" {
		return nil
	}
	for _, requirement := range requirements {
		if requirement.Class == action.Trust {
			return nil
		}
	}

	return model.NewError(
		model.ErrorTrust,
		"The Composer action trust class is absent from the approved plan.",
	)
}
