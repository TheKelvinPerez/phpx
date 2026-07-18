package state

import (
	"sort"

	"github.com/elefantephp/elefante/internal/model"
)

const TrustSchemaVersion = "elefante.trust/v1"

type TrustApproval struct {
	Class       model.TrustClass `json:"class"`
	Fingerprint string           `json:"fingerprint"`
}

type TrustRecord struct {
	SchemaVersion   string          `json:"schema_version"`
	ProjectIdentity string          `json:"project_identity"`
	Approvals       []TrustApproval `json:"approvals"`
}

func NewTrustRecord(projectIdentity string) TrustRecord {
	return TrustRecord{
		SchemaVersion:   TrustSchemaVersion,
		ProjectIdentity: projectIdentity,
		Approvals:       []TrustApproval{},
	}
}

func (record *TrustRecord) Approve(requirements []model.TrustRequirement) {
	approved := make(
		map[model.TrustClass]map[string]struct{},
		len(record.Approvals),
	)
	for _, approval := range record.Approvals {
		addApproval(approved, approval.Class, approval.Fingerprint)
	}
	for _, requirement := range requirements {
		addApproval(approved, requirement.Class, requirement.Fingerprint)
	}

	record.Approvals = record.Approvals[:0]
	for class, fingerprints := range approved {
		for fingerprint := range fingerprints {
			record.Approvals = append(record.Approvals, TrustApproval{
				Class:       class,
				Fingerprint: fingerprint,
			})
		}
	}
	sort.Slice(record.Approvals, func(left int, right int) bool {
		if record.Approvals[left].Class != record.Approvals[right].Class {
			return record.Approvals[left].Class < record.Approvals[right].Class
		}

		return record.Approvals[left].Fingerprint <
			record.Approvals[right].Fingerprint
	})
}

func (record TrustRecord) Missing(
	requirements []model.TrustRequirement,
) []model.TrustRequirement {
	approved := make(
		map[model.TrustClass]map[string]struct{},
		len(record.Approvals),
	)
	for _, approval := range record.Approvals {
		addApproval(approved, approval.Class, approval.Fingerprint)
	}

	var missing []model.TrustRequirement
	for _, requirement := range requirements {
		fingerprints := approved[requirement.Class]
		if _, exists := fingerprints[requirement.Fingerprint]; exists {
			continue
		}
		missing = append(missing, requirement)
	}

	return missing
}

func addApproval(
	approved map[model.TrustClass]map[string]struct{},
	class model.TrustClass,
	fingerprint string,
) {
	if approved[class] == nil {
		approved[class] = make(map[string]struct{})
	}
	approved[class][fingerprint] = struct{}{}
}
