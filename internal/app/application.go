package app

import (
	"context"

	"github.com/elefantephp/elefante/internal/model"
)

type Dependencies struct {
	Build model.BuildInfo
}

type Application struct {
	build model.BuildInfo
}

func New(dependencies Dependencies) *Application {
	return &Application{
		build: dependencies.Build,
	}
}

func (application *Application) Version(context.Context) model.BuildInfo {
	return application.build
}
