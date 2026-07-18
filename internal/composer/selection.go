package composer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/elefantephp/elefante/internal/constraints"
	"github.com/elefantephp/elefante/internal/model"
)

const (
	SourceManaged  = "managed"
	SourceProvider = "provider"
	SourceSystem   = "system"
)

func SelectExecutable(
	candidates []model.ComposerObservation,
	constraint string,
) (model.ComposerObservation, error) {
	compatible := make([]model.ComposerObservation, 0, len(candidates))
	for _, candidate := range candidates {
		matches := true
		var err error
		if strings.TrimSpace(constraint) != "" {
			matches, err = constraints.Satisfies(
				candidate.Version,
				constraint,
			)
			if err != nil {
				return model.ComposerObservation{}, fmt.Errorf(
					"evaluate Composer constraint %q: %w",
					constraint,
					err,
				)
			}
		}
		if matches {
			compatible = append(compatible, candidate)
		}
	}
	if len(compatible) == 0 {
		return model.ComposerObservation{}, model.NewError(
			model.ErrorRequirements,
			"No compatible Composer executable is available.",
		)
	}

	sort.Slice(compatible, func(left int, right int) bool {
		leftPriority := sourcePriority(compatible[left].Source)
		rightPriority := sourcePriority(compatible[right].Source)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if compatible[left].Version != compatible[right].Version {
			leftVersion, leftErr := constraints.NormalizeVersion(
				compatible[left].Version,
			)
			rightVersion, rightErr := constraints.NormalizeVersion(
				compatible[right].Version,
			)
			if leftErr == nil && rightErr == nil {
				return leftVersion.Compare(rightVersion) > 0
			}

			return compatible[left].Version > compatible[right].Version
		}
		if compatible[left].Identity != compatible[right].Identity {
			return compatible[left].Identity < compatible[right].Identity
		}

		return compatible[left].Path < compatible[right].Path
	})

	return compatible[0], nil
}

func sourcePriority(source string) int {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case SourceManaged, "elefante_managed":
		return 1
	case SourceSystem:
		return 3
	default:
		return 2
	}
}
