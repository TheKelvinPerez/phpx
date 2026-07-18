package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/state"
)

type SyncState interface {
	SaveJournal(string, state.ActionJournal) error
	LoadJournal(string) (state.ActionJournal, bool, error)
	SaveEnvironment(string, state.EnvironmentRecord) error
	LoadTrust(string) (state.TrustRecord, bool, error)
	SaveTrust(string, state.TrustRecord) error
}

type SyncLock interface {
	Release() error
}

type AcquireSyncLock func(string) (SyncLock, error)

type SyncActionExecution struct {
	Analysis       Analysis
	Action         model.PlanAction
	NonInteractive bool
}

type SyncActionResult struct {
	Outputs      []model.ActionOutput
	Compensation *SyncCompensation
}

type SyncCompensation struct {
	Safe   bool
	Action model.PlanAction
}

type SyncCompensationExecution struct {
	Analysis       Analysis
	OriginalAction model.PlanAction
	Compensation   SyncCompensation
}

type ExecuteSyncAction func(
	context.Context,
	SyncActionExecution,
) (SyncActionResult, error)

type CompensateSyncAction func(
	context.Context,
	SyncCompensationExecution,
) error

type SyncEngineDependencies struct {
	State            SyncState
	AcquireLock      AcquireSyncLock
	ExecuteAction    ExecuteSyncAction
	CompensateAction CompensateSyncAction
}

type SyncEngine struct {
	state            SyncState
	acquireLock      AcquireSyncLock
	executeAction    ExecuteSyncAction
	compensateAction CompensateSyncAction
}

type SyncExecution struct {
	Analysis       Analysis
	NonInteractive bool
	TrustApproved  bool
}

type SyncResult struct {
	CompletedActions []string
}

func NewSyncEngine(dependencies SyncEngineDependencies) *SyncEngine {
	return &SyncEngine{
		state:            dependencies.State,
		acquireLock:      dependencies.AcquireLock,
		executeAction:    dependencies.ExecuteAction,
		compensateAction: dependencies.CompensateAction,
	}
}

type completedSyncAction struct {
	action model.PlanAction
	result SyncActionResult
}

func (engine *SyncEngine) Apply(
	ctx context.Context,
	execution SyncExecution,
) (result SyncResult, resultErr error) {
	if engine == nil ||
		engine.state == nil ||
		engine.acquireLock == nil ||
		engine.executeAction == nil {
		return SyncResult{}, model.NewError(
			model.ErrorInternal,
			"The synchronization engine is not fully configured.",
		)
	}
	identity := strings.TrimSpace(
		execution.Analysis.Facts.Identity.IdentityKey,
	)
	if identity == "" {
		identity = strings.TrimSpace(
			execution.Analysis.Plan.Project.IdentityKey,
		)
	}
	if identity == "" {
		return SyncResult{}, model.NewError(
			model.ErrorState,
			"The synchronization plan has no project identity.",
		)
	}

	lock, err := engine.acquireLock(identity)
	if err != nil {
		return SyncResult{}, err
	}
	defer func() {
		if err := lock.Release(); err != nil && resultErr == nil {
			result = SyncResult{}
			resultErr = model.WrapError(
				model.ErrorState,
				"Could not release the synchronization lock.",
				err,
			)
		}
	}()

	trustRecord := state.NewTrustRecord(identity)
	var missingTrust []model.TrustRequirement
	if len(execution.Analysis.Plan.Trust) > 0 {
		loadedTrust, exists, err := engine.state.LoadTrust(identity)
		if err != nil {
			return SyncResult{}, err
		}
		if exists {
			trustRecord = loadedTrust
		}
		missingTrust = trustRecord.Missing(execution.Analysis.Plan.Trust)
		if len(missingTrust) > 0 && !execution.TrustApproved {
			return SyncResult{}, missingTrustError(missingTrust)
		}
	}

	journal, exists, err := engine.state.LoadJournal(identity)
	if err != nil {
		return SyncResult{}, err
	}
	if !exists || journal.PlanDigest != execution.Analysis.Plan.Digest {
		journal = state.NewActionJournal(
			identity,
			execution.Analysis.Plan.Digest,
			execution.Analysis.Plan.Actions,
		)
	} else {
		if journal.Status == state.JournalSucceeded {
			if len(missingTrust) > 0 {
				trustRecord.Approve(execution.Analysis.Plan.Trust)
				if err := engine.state.SaveTrust(
					identity,
					trustRecord,
				); err != nil {
					return SyncResult{}, err
				}
			}
			for _, action := range journal.Actions {
				result.CompletedActions = append(
					result.CompletedActions,
					action.ID,
				)
			}

			return result, nil
		}
		if err := journal.PrepareRetry(
			execution.Analysis.Plan.Actions,
		); err != nil {
			return SyncResult{}, syncJournalError("", err)
		}
		for _, action := range journal.Actions {
			if action.Status == state.ActionCompleted {
				result.CompletedActions = append(
					result.CompletedActions,
					action.ID,
				)
			}
		}
	}
	if err := engine.state.SaveJournal(identity, journal); err != nil {
		return SyncResult{}, err
	}
	if len(missingTrust) > 0 {
		trustRecord.Approve(execution.Analysis.Plan.Trust)
		if err := engine.state.SaveTrust(identity, trustRecord); err != nil {
			return SyncResult{}, err
		}
	}
	var completedThisRun []completedSyncAction
	for actionIndex, action := range execution.Analysis.Plan.Actions {
		if journal.Actions[actionIndex].Status == state.ActionCompleted {
			continue
		}
		var actionResult SyncActionResult
		var err error
		if action.Kind == model.ActionRecordState {
			completedActions := append(
				append([]string(nil), result.CompletedActions...),
				action.ID,
			)
			err = engine.state.SaveEnvironment(
				identity,
				state.NewEnvironmentRecord(
					identity,
					execution.Analysis.Plan,
					completedActions,
				),
			)
			actionResult.Outputs = append(
				[]model.ActionOutput(nil),
				action.ExpectedOutputs...,
			)
		} else {
			actionResult, err = engine.executeAction(
				ctx,
				SyncActionExecution{
					Analysis:       execution.Analysis,
					Action:         action,
					NonInteractive: execution.NonInteractive,
				},
			)
		}
		if err != nil {
			if journalErr := journal.Fail(action.ID, err.Error()); journalErr != nil {
				return result, syncJournalError(action.ID, journalErr)
			}
			if journalErr := engine.state.SaveJournal(
				identity,
				journal,
			); journalErr != nil {
				return result, journalErr
			}
			compensated, manualRecovery, compensationErr := engine.compensate(
				ctx,
				execution.Analysis,
				identity,
				&journal,
				completedThisRun,
			)
			if compensationErr != nil {
				return result, compensationErr
			}

			return result, syncActionError(
				execution.Analysis.Plan,
				action,
				result.CompletedActions,
				execution.Analysis.Plan.Actions[actionIndex+1:],
				compensated,
				manualRecovery,
				journal,
				err,
			)
		}
		if err := journal.CompleteWithOutputs(
			action.ID,
			actionResult.Outputs,
		); err != nil {
			return SyncResult{}, syncJournalError(action.ID, err)
		}
		if err := engine.state.SaveJournal(identity, journal); err != nil {
			return SyncResult{}, err
		}
		result.CompletedActions = append(
			result.CompletedActions,
			action.ID,
		)
		completedThisRun = append(completedThisRun, completedSyncAction{
			action: action,
			result: actionResult,
		})
	}
	if err := journal.Succeed(); err != nil {
		return SyncResult{}, syncJournalError("", err)
	}
	if err := engine.state.SaveJournal(identity, journal); err != nil {
		return SyncResult{}, err
	}

	return result, nil
}

func missingTrustError(
	requirements []model.TrustRequirement,
) *model.Error {
	values := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		values = append(
			values,
			string(requirement.Class)+"="+requirement.Fingerprint,
		)
	}
	commandError := model.NewError(
		model.ErrorTrust,
		"Composer execution requires approval for the current trust fingerprint.",
	).WithHint("Review the exact plan and approve it before synchronization.")
	commandError.Details = []model.ErrorDetail{
		{Name: "missing_trust", Value: strings.Join(values, ",")},
	}

	return commandError
}

func (engine *SyncEngine) compensate(
	ctx context.Context,
	analysis Analysis,
	identity string,
	journal *state.ActionJournal,
	completed []completedSyncAction,
) ([]string, []string, error) {
	var compensated []string
	var manualRecovery []string
	for index := len(completed) - 1; index >= 0; index-- {
		completedAction := completed[index]
		compensation := completedAction.result.Compensation
		if !completedAction.action.Reversible ||
			compensation == nil ||
			!compensation.Safe ||
			engine.compensateAction == nil {
			if completedAction.action.Effect != model.EffectRead {
				manualRecovery = append(
					manualRecovery,
					completedAction.action.ID,
				)
			}
			continue
		}
		if err := engine.compensateAction(
			ctx,
			SyncCompensationExecution{
				Analysis:       analysis,
				OriginalAction: completedAction.action,
				Compensation:   *compensation,
			},
		); err != nil {
			manualRecovery = append(
				manualRecovery,
				completedAction.action.ID,
			)
			continue
		}
		if err := journal.Compensate(completedAction.action.ID); err != nil {
			return nil, nil, syncJournalError(
				completedAction.action.ID,
				err,
			)
		}
		if err := engine.state.SaveJournal(identity, *journal); err != nil {
			return nil, nil, err
		}
		compensated = append(compensated, completedAction.action.ID)
	}

	return compensated, manualRecovery, nil
}

func syncActionError(
	plan model.Plan,
	failedAction model.PlanAction,
	completedActions []string,
	notAttempted []model.PlanAction,
	compensatedActions []string,
	manualRecoveryActions []string,
	journal state.ActionJournal,
	cause error,
) *model.Error {
	pendingIDs := make([]string, 0, len(notAttempted))
	for _, action := range notAttempted {
		pendingIDs = append(pendingIDs, action.ID)
	}
	commandError := model.WrapError(
		model.ErrorSync,
		"Synchronization stopped after an action failed.",
		cause,
	).WithRetryable(true)
	commandError.Details = []model.ErrorDetail{
		{
			Name:  "completed_actions",
			Value: strings.Join(completedActions, ","),
		},
		{Name: "failed_action", Value: failedAction.ID},
		{
			Name:  "not_attempted_actions",
			Value: strings.Join(pendingIDs, ","),
		},
		{
			Name:  "observed_partial_state",
			Value: observedPartialState(journal),
		},
		{
			Name:  "safe_retry_command",
			Value: "elefante sync --approve-plan " + plan.Digest,
		},
		{
			Name:  "compensated_actions",
			Value: strings.Join(compensatedActions, ","),
		},
		{
			Name:  "manual_recovery_actions",
			Value: strings.Join(manualRecoveryActions, ","),
		},
		{
			Name:  "manual_recovery",
			Value: "Review the failed action and its partial state before retrying.",
		},
	}

	return commandError
}

func observedPartialState(journal state.ActionJournal) string {
	var values []string
	for _, action := range journal.Actions {
		if action.Status != state.ActionCompleted || action.Compensated {
			continue
		}
		for _, output := range action.Outputs {
			values = append(
				values,
				action.ID+":"+output.Name+"="+output.Value,
			)
		}
	}

	return strings.Join(values, ",")
}

func syncJournalError(actionID string, cause error) *model.Error {
	message := "The synchronization journal could not advance."
	if actionID != "" {
		message = fmt.Sprintf(
			"The synchronization journal could not advance action %q.",
			actionID,
		)
	}

	return model.WrapError(model.ErrorState, message, cause)
}
