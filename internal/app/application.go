package app

import (
	"context"
	"sort"
	"strings"

	"github.com/elefantephp/elefante/internal/discovery"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/plan"
	"github.com/elefantephp/elefante/internal/providers"
	"github.com/elefantephp/elefante/internal/security"
)

type DiscoverProject func(context.Context, discovery.Request) (model.ProjectFacts, error)
type ExecuteApprovedPlan func(context.Context, Analysis) error

type Dependencies struct {
	Build               model.BuildInfo
	DiscoverProject     DiscoverProject
	Providers           []providers.Provider
	ExecuteApprovedPlan ExecuteApprovedPlan
}

type Application struct {
	build               model.BuildInfo
	discoverProject     DiscoverProject
	providers           []providers.Provider
	executeApprovedPlan ExecuteApprovedPlan
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
}

type Analysis struct {
	Facts        model.ProjectFacts          `json:"facts"`
	Observations []model.ProviderObservation `json:"observations"`
	Plan         model.Plan                  `json:"plan"`
}

func New(dependencies Dependencies) *Application {
	discoverProject := dependencies.DiscoverProject
	if discoverProject == nil {
		discoverProject = discovery.Discover
	}

	executeApprovedPlan := dependencies.ExecuteApprovedPlan
	if executeApprovedPlan == nil {
		executeApprovedPlan = func(context.Context, Analysis) error {
			return model.NewError(
				model.ErrorSync,
				"Synchronization execution is not available in this build phase.",
			).WithHint(
				"Use elefante plan to review the synchronization work.",
			)
		}
	}

	return &Application{
		build:               dependencies.Build,
		discoverProject:     discoverProject,
		providers:           append([]providers.Provider(nil), dependencies.Providers...),
		executeApprovedPlan: executeApprovedPlan,
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
	if err := application.executeApprovedPlan(ctx, analysis); err != nil {
		return analysis, err
	}

	return analysis, nil
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
