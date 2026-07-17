package app

import (
	"context"

	"github.com/elefantephp/elefante/internal/discovery"
	"github.com/elefantephp/elefante/internal/model"
)

type DiscoverProject func(context.Context, discovery.Request) (model.ProjectFacts, error)

type Dependencies struct {
	Build           model.BuildInfo
	DiscoverProject DiscoverProject
}

type Application struct {
	build           model.BuildInfo
	discoverProject DiscoverProject
}

func New(dependencies Dependencies) *Application {
	discoverProject := dependencies.DiscoverProject
	if discoverProject == nil {
		discoverProject = discovery.Discover
	}

	return &Application{
		build:           dependencies.Build,
		discoverProject: discoverProject,
	}
}

func (application *Application) Version(context.Context) model.BuildInfo {
	return application.build
}

func (application *Application) Doctor(
	ctx context.Context,
	projectPath string,
) (model.ProjectFacts, error) {
	return application.discoverProject(ctx, discovery.Request{
		StartPath: projectPath,
	})
}
