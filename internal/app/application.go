package app

import (
	"context"
	"sort"

	"github.com/elefantephp/elefante/internal/discovery"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/plan"
	"github.com/elefantephp/elefante/internal/providers"
)

type DiscoverProject func(context.Context, discovery.Request) (model.ProjectFacts, error)

type Dependencies struct {
	Build           model.BuildInfo
	DiscoverProject DiscoverProject
	Providers       []providers.Provider
}

type Application struct {
	build           model.BuildInfo
	discoverProject DiscoverProject
	providers       []providers.Provider
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

	return &Application{
		build:           dependencies.Build,
		discoverProject: discoverProject,
		providers:       append([]providers.Provider(nil), dependencies.Providers...),
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

	builtPlan, err := plan.Build(plan.Request{
		Operation:    request.Operation,
		Facts:        facts,
		Observations: observations,
		Provider:     request.Provider,
		Policy:       request.Policy,
	})
	if err != nil {
		return Analysis{}, err
	}

	return Analysis{
		Facts:        facts,
		Observations: observations,
		Plan:         builtPlan,
	}, nil
}
