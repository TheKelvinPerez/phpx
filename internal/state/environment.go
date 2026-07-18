package state

import (
	"github.com/elefantephp/elefante/internal/model"
)

const EnvironmentSchemaVersion = "elefante.environment/v1"

type EnvironmentRecord struct {
	SchemaVersion    string                   `json:"schema_version"`
	ProjectIdentity  string                   `json:"project_identity"`
	PlanDigest       string                   `json:"plan_digest"`
	Provider         model.ProviderSelection  `json:"provider"`
	Inputs           []model.InputFingerprint `json:"inputs"`
	CompletedActions []string                 `json:"completed_actions"`
}

func NewEnvironmentRecord(
	projectIdentity string,
	plan model.Plan,
	completedActions []string,
) EnvironmentRecord {
	return EnvironmentRecord{
		SchemaVersion:    EnvironmentSchemaVersion,
		ProjectIdentity:  projectIdentity,
		PlanDigest:       plan.Digest,
		Provider:         plan.Provider,
		Inputs:           append([]model.InputFingerprint(nil), plan.Inputs...),
		CompletedActions: append([]string(nil), completedActions...),
	}
}
