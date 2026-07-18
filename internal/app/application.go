package app

import (
	"context"
	"io"
	"sort"
	"strings"

	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/discovery"
	"github.com/elefantephp/elefante/internal/executor"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/plan"
	"github.com/elefantephp/elefante/internal/providers"
	"github.com/elefantephp/elefante/internal/security"
)

type DiscoverProject func(context.Context, discovery.Request) (model.ProjectFacts, error)
type ExecuteApprovedPlan func(context.Context, Analysis) error
type ApplySynchronization func(
	context.Context,
	SyncExecution,
) (SyncResult, error)
type ExecuteProcess func(
	context.Context,
	executor.Command,
	executor.Streams,
) (executor.Result, error)

type ManagedComposer interface {
	Resolve(
		context.Context,
		composer.ResolveRequest,
	) (composer.Release, error)
	Observation(
		composer.Release,
	) (model.ComposerObservation, error)
}

type Dependencies struct {
	Build                model.BuildInfo
	DiscoverProject      DiscoverProject
	Providers            []providers.Provider
	ManagedComposer      ManagedComposer
	ExecuteApprovedPlan  ExecuteApprovedPlan
	ApplySynchronization ApplySynchronization
	ExecuteProcess       ExecuteProcess
}

type Application struct {
	build                model.BuildInfo
	discoverProject      DiscoverProject
	providers            []providers.Provider
	managedComposer      ManagedComposer
	applySynchronization ApplySynchronization
	executeProcess       ExecuteProcess
}

type DoctorRequest struct {
	ProjectPath string
	ConfigPath  string
	Provider    string
}

type PlanRequest struct {
	ProjectPath string
	ConfigPath  string
	Provider    string
	Offline     bool
	Frozen      bool
}

type SyncRequest struct {
	ProjectPath    string
	ConfigPath     string
	Provider       string
	Offline        bool
	Frozen         bool
	NonInteractive bool
	Yes            bool
	ApprovedPlan   string
	Confirmed      bool
	Input          io.Reader
	Output         io.Writer
	Error          io.Writer
}

type Analysis struct {
	Facts        model.ProjectFacts          `json:"facts"`
	Observations []model.ProviderObservation `json:"observations"`
	Plan         model.Plan                  `json:"plan"`
}

type RunRequest struct {
	ProjectPath string
	ConfigPath  string
	Provider    string
	Arguments   []string
	Input       io.Reader
	Output      io.Writer
	Error       io.Writer
}

type RunResult struct {
	Exit      model.Exit
	Signal    string
	Cancelled bool
}

func New(dependencies Dependencies) *Application {
	discoverProject := dependencies.DiscoverProject
	if discoverProject == nil {
		discoverProject = discovery.Discover
	}

	applySynchronization := dependencies.ApplySynchronization
	if applySynchronization == nil && dependencies.ExecuteApprovedPlan != nil {
		applySynchronization = func(
			ctx context.Context,
			execution SyncExecution,
		) (SyncResult, error) {
			err := dependencies.ExecuteApprovedPlan(ctx, execution.Analysis)

			return SyncResult{}, err
		}
	}
	if applySynchronization == nil {
		applySynchronization = func(
			context.Context,
			SyncExecution,
		) (SyncResult, error) {
			return SyncResult{}, model.NewError(
				model.ErrorSync,
				"Synchronization execution is not available in this build phase.",
			).WithHint(
				"Use elefante plan to review the synchronization work.",
			)
		}
	}

	return &Application{
		build:                dependencies.Build,
		discoverProject:      discoverProject,
		providers:            append([]providers.Provider(nil), dependencies.Providers...),
		managedComposer:      dependencies.ManagedComposer,
		applySynchronization: applySynchronization,
		executeProcess:       dependencies.ExecuteProcess,
	}
}

func (application *Application) Version(context.Context) model.BuildInfo {
	return application.build
}

func (application *Application) Doctor(
	ctx context.Context,
	request DoctorRequest,
) (Analysis, error) {
	return application.analyze(ctx, analysisRequest{
		ProjectPath: request.ProjectPath,
		ConfigPath:  request.ConfigPath,
		Provider:    request.Provider,
		Operation:   model.OperationDoctor,
		Policy: model.PlanPolicy{
			Offline: true,
			Frozen:  true,
		},
	})
}

func (application *Application) Plan(
	ctx context.Context,
	request PlanRequest,
) (Analysis, error) {
	return application.analyze(ctx, analysisRequest{
		ProjectPath: request.ProjectPath,
		ConfigPath:  request.ConfigPath,
		Provider:    request.Provider,
		Operation:   model.OperationSync,
		Policy: model.PlanPolicy{
			Offline: request.Offline,
			Frozen:  request.Frozen,
		},
	})
}

func (application *Application) Sync(
	ctx context.Context,
	request SyncRequest,
) (Analysis, error) {
	analysis, err := application.analyze(ctx, analysisRequest{
		ProjectPath: request.ProjectPath,
		ConfigPath:  request.ConfigPath,
		Provider:    request.Provider,
		Operation:   model.OperationSync,
		Policy: model.PlanPolicy{
			Offline: request.Offline,
			Frozen:  request.Frozen,
		},
	})
	if err != nil {
		return Analysis{}, err
	}
	if err := BlockingPlanError(analysis.Plan); err != nil {
		return analysis, err
	}
	if err := security.AuthorizePlan(
		analysis.Plan,
		security.ApprovalOptions{
			Yes:          request.Yes,
			ApprovedPlan: request.ApprovedPlan,
			Confirmed:    request.Confirmed,
		},
	); err != nil {
		return analysis, err
	}
	if _, err := application.applySynchronization(ctx, SyncExecution{
		Analysis:       analysis,
		NonInteractive: request.NonInteractive,
		TrustApproved: request.Yes ||
			request.ApprovedPlan != "" ||
			request.Confirmed,
		Input:  request.Input,
		Output: request.Output,
		Error:  request.Error,
	}); err != nil {
		return analysis, err
	}

	return analysis, nil
}

func (application *Application) Run(
	ctx context.Context,
	request RunRequest,
) (RunResult, error) {
	if len(request.Arguments) == 0 ||
		strings.TrimSpace(request.Arguments[0]) == "" {
		return RunResult{}, model.NewError(
			model.ErrorUsage,
			"Run requires a child executable after the command separator.",
		)
	}
	analysis, err := application.analyze(ctx, analysisRequest{
		ProjectPath: request.ProjectPath,
		ConfigPath:  request.ConfigPath,
		Provider:    request.Provider,
		Operation:   model.OperationDoctor,
		Policy: model.PlanPolicy{
			Offline: true,
			Frozen:  true,
		},
	})
	if err != nil {
		return RunResult{}, err
	}
	if err := BlockingPlanError(analysis.Plan); err != nil {
		return RunResult{}, err
	}
	selected, err := application.selectedProvider(
		analysis.Plan.Provider.Name,
	)
	if err != nil {
		return RunResult{}, err
	}
	specification, err := selected.ExecutionSpec(
		ctx,
		providers.ExecutionRequest{
			Executable: request.Arguments[0],
			Arguments: append(
				[]string(nil),
				request.Arguments[1:]...,
			),
			WorkingDirectory: analysis.Facts.Identity.ApplicationRoot,
		},
	)
	if err != nil {
		return RunResult{}, err
	}
	if application.executeProcess == nil {
		return RunResult{}, model.NewError(
			model.ErrorInternal,
			"Command execution is not configured.",
		)
	}
	executed, err := application.executeProcess(
		ctx,
		executor.Command{
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
		},
		executor.Streams{
			Input:  request.Input,
			Output: request.Output,
			Error:  request.Error,
		},
	)
	result := RunResult{}
	if executed.Started {
		result = RunResult{
			Exit: model.Exit{
				Origin: model.ExitOriginChild,
				Code:   executed.ExitCode,
			},
			Signal:    executed.Signal,
			Cancelled: executed.Cancelled,
		}
	}
	if err != nil {
		commandError := model.WrapError(
			model.ErrorProvider,
			"Could not execute the command in the selected environment.",
			err,
		)
		commandError.Provider = analysis.Plan.Provider.Name

		return result, commandError
	}
	if !executed.Started {
		return RunResult{}, model.NewError(
			model.ErrorInternal,
			"The process executor returned without starting the child command.",
		)
	}

	return result, nil
}

type analysisRequest struct {
	ProjectPath string
	ConfigPath  string
	Provider    string
	Operation   model.Operation
	Policy      model.PlanPolicy
}

func (application *Application) analyze(
	ctx context.Context,
	request analysisRequest,
) (Analysis, error) {
	facts, err := application.discoverProject(ctx, discovery.Request{
		StartPath:  request.ProjectPath,
		ConfigPath: request.ConfigPath,
	})
	if err != nil {
		return Analysis{}, err
	}

	registered := append(
		[]providers.Provider(nil),
		application.providers...,
	)
	sort.SliceStable(registered, func(left int, right int) bool {
		return registered[left].Name() < registered[right].Name()
	})

	observations := make(
		[]model.ProviderObservation,
		0,
		len(registered),
	)
	for _, provider := range registered {
		observation, err := provider.Inspect(ctx, providers.InspectRequest{
			Project: facts.Identity,
			Offline: request.Policy.Offline,
		})
		if err != nil {
			return Analysis{}, err
		}
		observations = append(observations, observation)
	}

	planRequest := plan.Request{
		Operation:    request.Operation,
		Facts:        facts,
		Observations: observations,
		Provider:     request.Provider,
		Policy:       request.Policy,
	}
	builtPlan, err := plan.Build(planRequest)
	if err != nil {
		return Analysis{}, err
	}
	if application.managedComposer != nil &&
		request.Operation == model.OperationSync {
		selectedIndex := providerObservationIndex(
			observations,
			builtPlan.Provider.Name,
		)
		phpVersion := selectedPHPVersion(builtPlan)
		constraint := strings.TrimSpace(
			facts.Configuration.Composer.Constraint,
		)
		if selectedIndex >= 0 &&
			phpVersion != "" &&
			(constraint != "" ||
				len(observations[selectedIndex].Composer) == 0) {
			release, err := application.managedComposer.Resolve(
				ctx,
				composer.ResolveRequest{
					Constraint: constraint,
					PHPVersion: phpVersion,
					Offline:    request.Policy.Offline,
				},
			)
			if err != nil {
				return Analysis{}, err
			}
			managedObservation, err := application.managedComposer.Observation(
				release,
			)
			if err != nil {
				return Analysis{}, err
			}
			observations[selectedIndex].Composer = append(
				observations[selectedIndex].Composer,
				managedObservation,
			)
			planRequest.Observations = observations
			builtPlan, err = plan.Build(planRequest)
			if err != nil {
				return Analysis{}, err
			}
		}
	}
	if request.Operation != model.OperationDoctor &&
		builtPlan.Provider.Name != "" {
		for index, provider := range registered {
			if !strings.EqualFold(
				strings.TrimSpace(provider.Name()),
				builtPlan.Provider.Name,
			) {
				continue
			}
			providerPlan, err := provider.Plan(
				ctx,
				providers.ProviderPlanRequest{
					Facts:       facts,
					Observation: observations[index],
					Policy:      request.Policy,
				},
			)
			if err != nil {
				return Analysis{}, err
			}
			planRequest.ProviderPlans = map[string]providers.ProviderPlan{
				builtPlan.Provider.Name: providerPlan,
			}
			builtPlan, err = plan.Build(planRequest)
			if err != nil {
				return Analysis{}, err
			}
			break
		}
	}

	return Analysis{
		Facts:        facts,
		Observations: observations,
		Plan:         builtPlan,
	}, nil
}

func providerObservationIndex(
	observations []model.ProviderObservation,
	name string,
) int {
	for index, observation := range observations {
		if strings.EqualFold(
			strings.TrimSpace(observation.Provider),
			strings.TrimSpace(name),
		) {
			return index
		}
	}

	return -1
}

func (application *Application) selectedProvider(
	name string,
) (providers.Provider, error) {
	for _, candidate := range application.providers {
		if strings.EqualFold(
			strings.TrimSpace(candidate.Name()),
			strings.TrimSpace(name),
		) {
			return candidate, nil
		}
	}
	commandError := model.NewError(
		model.ErrorProvider,
		"The selected execution provider is not registered.",
	)
	commandError.Provider = name

	return nil, commandError
}

func selectedPHPVersion(builtPlan model.Plan) string {
	for _, requirement := range builtPlan.Requirements {
		if requirement.Kind == model.RequirementPHP &&
			requirement.SelectedValue != "" {
			return requirement.SelectedValue
		}
	}

	return ""
}

func BlockingPlanError(builtPlan model.Plan) error {
	for _, diagnostic := range builtPlan.Diagnostics {
		if diagnostic.Severity != model.SeverityError {
			continue
		}
		code := model.ErrorRequirements
		if strings.HasPrefix(diagnostic.Code, "ELEFANTE_OFFLINE") ||
			strings.HasPrefix(diagnostic.Code, "ELEFANTE_NETWORK") {
			code = model.ErrorNetwork
		} else if strings.HasPrefix(diagnostic.Code, "ELEFANTE_PROVIDER") ||
			strings.HasPrefix(diagnostic.Code, "ELEFANTE_NATIVE") {
			code = model.ErrorProvider
		}
		commandError := model.NewError(
			code,
			"Project analysis found a blocking diagnostic.",
		)
		commandError.Detail = diagnostic.Message
		commandError.Hint = diagnostic.Hint
		commandError.Sources = append(
			[]model.SourceReference(nil),
			diagnostic.Sources...,
		)
		commandError.Provider = diagnostic.Provider

		return commandError
	}

	return nil
}
