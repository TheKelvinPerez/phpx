package cache

import (
	"strings"

	"github.com/elefantephp/elefante/internal/model"
)

type Command string

const (
	CommandDoctor Command = "doctor"
	CommandPlan   Command = "plan"
	CommandSync   Command = "sync"
	CommandRun    Command = "run"
	CommandTool   Command = "tool"
)

type Artifact struct {
	Identity string `json:"identity"`
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
}

type NetworkPolicy struct {
	Command Command
	Offline bool
}

type NetworkRequest struct {
	Requirement    model.NetworkRequirement
	ApprovedSource bool
	Refresh        bool
	Substitute     *Artifact
}

type NetworkDecision struct {
	UseNetwork bool
	Artifact   *Artifact
}

func (policy NetworkPolicy) Resolve(
	request NetworkRequest,
) (NetworkDecision, error) {
	if request.Requirement == model.NetworkNone {
		return NetworkDecision{}, nil
	}
	if request.Substitute != nil && !request.Refresh {
		if err := validateArtifact(*request.Substitute); err != nil {
			return NetworkDecision{}, err
		}
		copy := *request.Substitute

		return NetworkDecision{Artifact: &copy}, nil
	}
	if policy.Offline {
		return NetworkDecision{}, networkError(
			"Offline mode requires a verified local artifact.",
			"Populate the cache while online, then retry with --offline.",
		)
	}

	switch policy.Command {
	case CommandDoctor:
		return NetworkDecision{}, networkError(
			"The doctor command does not permit network access.",
			"Use locally available provider and project information.",
		)
	case CommandPlan:
		if request.Requirement != model.NetworkRead {
			return NetworkDecision{}, networkError(
				"The plan command permits read only metadata requests.",
				"Run sync with explicit approval for required downloads.",
			)
		}
	case CommandSync:
		if !request.ApprovedSource {
			return NetworkDecision{}, networkError(
				"The synchronization source is not represented in the approved plan.",
				"Build and approve a plan that contains this source.",
			)
		}
	case CommandRun:
		return NetworkDecision{}, networkError(
			"The run command does not initiate network access.",
			"Network behavior belongs to the explicitly launched child process.",
		)
	case CommandTool:
	default:
		return NetworkDecision{}, networkError(
			"The command does not have a network policy.",
			"Use a command with an explicit network contract.",
		)
	}

	return NetworkDecision{UseNetwork: true}, nil
}

func validateArtifact(artifact Artifact) error {
	if !strings.HasPrefix(artifact.Identity, "sha256:") ||
		!strings.HasPrefix(artifact.SHA256, "sha256:") ||
		strings.TrimSpace(artifact.Path) == "" {
		return model.NewError(
			model.ErrorArtifact,
			"The cached network substitute is not content identified.",
		).WithHint("Remove the invalid cache entry and reacquire it while online.")
	}

	return nil
}

func networkError(message string, hint string) *model.Error {
	return model.NewError(model.ErrorNetwork, message).WithHint(hint)
}
