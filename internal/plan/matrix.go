package plan

import (
	"github.com/elefantephp/elefante/internal/constraints"
	"github.com/elefantephp/elefante/internal/model"
)

const supportMatrixVersion = "phase1/v1"

var supportedPHPVersions = []string{
	"8.5.0",
	"8.4.0",
	"8.3.0",
}

var legacyPHPConstraints = []string{
	">=8.2 <8.3",
}

const supportedLaravelConstraint = ">=12 <14"
const legacyLaravelConstraint = ">=11 <12"

func supportedRuntimeTarget(
	requirements []model.Requirement,
) (string, bool, error) {
	for _, candidate := range supportedPHPVersions {
		satisfied, err := requirementsSatisfied(requirements, candidate)
		if err != nil {
			return "", false, err
		}
		if satisfied {
			return candidate, true, nil
		}
	}

	return "", false, nil
}

func legacyRuntime(
	group requirementGroup,
	selected string,
) (bool, error) {
	if group.kind != model.RequirementPHP {
		return false, nil
	}
	for _, constraint := range legacyPHPConstraints {
		matches, err := constraints.Satisfies(selected, constraint)
		if err != nil {
			return false, err
		}
		if matches {
			return true, nil
		}
	}

	return false, nil
}

func isLegacyLaravelConstraint(constraint string) bool {
	supported, err := constraints.Intersects(
		constraint,
		supportedLaravelConstraint,
	)
	if err != nil || supported {
		return false
	}
	legacy, err := constraints.Intersects(
		constraint,
		legacyLaravelConstraint,
	)

	return err == nil && legacy
}
