package state

import (
	"fmt"

	"github.com/elefantephp/elefante/internal/model"
)

const ActionJournalSchemaVersion = "elefante.action-journal/v1"

type JournalStatus string

const (
	JournalPending   JournalStatus = "pending"
	JournalFailed    JournalStatus = "failed"
	JournalSucceeded JournalStatus = "succeeded"
)

type ActionStatus string

const (
	ActionPending   ActionStatus = "pending"
	ActionCompleted ActionStatus = "completed"
	ActionFailed    ActionStatus = "failed"
)

type JournalAction struct {
	ID      string           `json:"id"`
	Kind    model.ActionKind `json:"kind"`
	Status  ActionStatus     `json:"status"`
	Failure string           `json:"failure,omitempty"`
}

type ActionJournal struct {
	SchemaVersion   string          `json:"schema_version"`
	ProjectIdentity string          `json:"project_identity"`
	PlanDigest      string          `json:"plan_digest"`
	Status          JournalStatus   `json:"status"`
	Actions         []JournalAction `json:"actions"`
}

func NewActionJournal(
	projectIdentity string,
	planDigest string,
	actions []model.PlanAction,
) ActionJournal {
	journalActions := make([]JournalAction, len(actions))
	for index, action := range actions {
		journalActions[index] = JournalAction{
			ID:     action.ID,
			Kind:   action.Kind,
			Status: ActionPending,
		}
	}

	return ActionJournal{
		SchemaVersion:   ActionJournalSchemaVersion,
		ProjectIdentity: projectIdentity,
		PlanDigest:      planDigest,
		Status:          JournalPending,
		Actions:         journalActions,
	}
}

func (journal *ActionJournal) Complete(actionID string) error {
	action, err := journal.action(actionID)
	if err != nil {
		return err
	}
	if action.Status != ActionPending {
		return fmt.Errorf(
			"action %q cannot complete from status %q",
			actionID,
			action.Status,
		)
	}
	action.Status = ActionCompleted
	action.Failure = ""

	return nil
}

func (journal *ActionJournal) Fail(actionID string, failure string) error {
	action, err := journal.action(actionID)
	if err != nil {
		return err
	}
	if action.Status != ActionPending {
		return fmt.Errorf(
			"action %q cannot fail from status %q",
			actionID,
			action.Status,
		)
	}
	action.Status = ActionFailed
	action.Failure = failure
	journal.Status = JournalFailed

	return nil
}

func (journal *ActionJournal) Succeed() error {
	for _, action := range journal.Actions {
		if action.Status != ActionCompleted {
			return fmt.Errorf(
				"action %q has status %q",
				action.ID,
				action.Status,
			)
		}
	}
	journal.Status = JournalSucceeded

	return nil
}

func (journal *ActionJournal) action(actionID string) (*JournalAction, error) {
	for index := range journal.Actions {
		if journal.Actions[index].ID == actionID {
			return &journal.Actions[index], nil
		}
	}

	return nil, fmt.Errorf("journal does not contain action %q", actionID)
}
