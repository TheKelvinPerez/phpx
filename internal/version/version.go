package version

import "github.com/elefantephp/elefante/internal/model"

const developmentVersion = "dev"

func Current() model.BuildInfo {
	return model.BuildInfo{
		Version: developmentVersion,
	}
}
