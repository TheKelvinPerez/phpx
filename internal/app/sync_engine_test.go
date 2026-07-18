package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/executor"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
	"github.com/elefantephp/elefante/internal/state"
)

func TestSyncEngineAppliesActionsInOrderWithJournalCheckpoints(t *testing.T) {
	journalStore := &fakeSyncState{}
	lock := &fakeSyncLock{}
	var executed []string
	engine := app.NewSyncEngine(app.SyncEngineDependencies{
		State: journalStore,
		AcquireLock: func(identity string) (app.SyncLock, error) {
			if identity != "sha256:project" {
				t.Fatalf("unexpected lock identity %q", identity)
			}
			lock.acquired = true

			return lock, nil
		},
		ExecuteAction: func(
			_ context.Context,
			execution app.SyncActionExecution,
		) (app.SyncActionResult, error) {
			if !lock.acquired || lock.released {
				t.Fatal("action executed outside the environment lock")
			}
			executed = append(executed, execution.Action.ID)

			return app.SyncActionResult{}, nil
		},
	})
	analysis := app.Analysis{
		Facts: model.ProjectFacts{
			Identity: model.ProjectIdentity{
				IdentityKey: "sha256:project",
			},
		},
		Plan: model.Plan{
			Digest: "sha256:approved",
			Actions: []model.PlanAction{
				{
					ID:   "01-prepare-provider",
					Kind: model.ActionPrepareProvider,
				},
				{
					ID:   "02-prepare-runtime",
					Kind: model.ActionPrepareRuntime,
				},
				{
					ID:   "03-install-dependencies",
					Kind: model.ActionInstallDependencies,
				},
			},
		},
	}

	result, err := engine.Apply(t.Context(), app.SyncExecution{
		Analysis: analysis,
	})
	if err != nil {
		t.Fatalf("apply approved synchronization: %v", err)
	}
	expectedOrder := []string{
		"01-prepare-provider",
		"02-prepare-runtime",
		"03-install-dependencies",
	}
	if !slices.Equal(executed, expectedOrder) {
		t.Fatalf("unexpected action order %#v", executed)
	}
	if !slices.Equal(result.CompletedActions, expectedOrder) {
		t.Fatalf("unexpected completed actions %#v", result.CompletedActions)
	}
	if !lock.released {
		t.Fatal("successful synchronization did not release its lock")
	}

	if len(journalStore.savedJournals) != 5 {
		t.Fatalf(
			"expected initial, per-action, and final journal writes, got %d",
			len(journalStore.savedJournals),
		)
	}
	initial := journalStore.savedJournals[0]
	if initial.Status != state.JournalPending {
		t.Fatalf("initial journal is not pending: %#v", initial)
	}
	for _, action := range initial.Actions {
		if action.Status != state.ActionPending {
			t.Fatalf("initial journal contains applied action %#v", action)
		}
	}
	for index := range expectedOrder {
		checkpoint := journalStore.savedJournals[index+1]
		for actionIndex, action := range checkpoint.Actions {
			expectedStatus := state.ActionPending
			if actionIndex <= index {
				expectedStatus = state.ActionCompleted
			}
			if action.Status != expectedStatus {
				t.Fatalf(
					"checkpoint %d action %d has status %q, expected %q",
					index+1,
					actionIndex,
					action.Status,
					expectedStatus,
				)
			}
		}
	}
	finalJournal := journalStore.savedJournals[4]
	if finalJournal.Status != state.JournalSucceeded {
		t.Fatalf("final journal did not succeed: %#v", finalJournal)
	}
}

func TestSyncEnginePersistsAndReportsPartialFailure(t *testing.T) {
	actionFailure := errors.New("provider action failed")
	journalStore := &fakeSyncState{}
	lock := &fakeSyncLock{}
	var executed []string
	engine := app.NewSyncEngine(app.SyncEngineDependencies{
		State: journalStore,
		AcquireLock: func(string) (app.SyncLock, error) {
			lock.acquired = true

			return lock, nil
		},
		ExecuteAction: func(
			_ context.Context,
			execution app.SyncActionExecution,
		) (app.SyncActionResult, error) {
			executed = append(executed, execution.Action.ID)
			if execution.Action.ID == "02-prepare-runtime" {
				return app.SyncActionResult{}, actionFailure
			}

			return app.SyncActionResult{
				Outputs: []model.ActionOutput{
					{Name: "provider_state", Value: "running"},
				},
			}, nil
		},
	})
	analysis := app.Analysis{
		Facts: model.ProjectFacts{
			Identity: model.ProjectIdentity{
				IdentityKey: "sha256:project",
			},
		},
		Plan: model.Plan{
			Digest: "sha256:approved",
			Actions: []model.PlanAction{
				{
					ID:   "01-prepare-provider",
					Kind: model.ActionPrepareProvider,
				},
				{
					ID:   "02-prepare-runtime",
					Kind: model.ActionPrepareRuntime,
				},
				{
					ID:   "03-install-dependencies",
					Kind: model.ActionInstallDependencies,
				},
			},
		},
	}

	result, err := engine.Apply(t.Context(), app.SyncExecution{
		Analysis: analysis,
	})
	if !errors.Is(err, actionFailure) {
		t.Fatalf("expected provider failure cause, got %v", err)
	}
	var commandError *model.Error
	if !errors.As(err, &commandError) ||
		commandError.Code != model.ErrorSync ||
		!commandError.Retryable {
		t.Fatalf("expected retryable synchronization error, got %#v", err)
	}
	if !slices.Equal(
		result.CompletedActions,
		[]string{"01-prepare-provider"},
	) {
		t.Fatalf("unexpected partial result %#v", result)
	}
	if !slices.Equal(
		executed,
		[]string{"01-prepare-provider", "02-prepare-runtime"},
	) {
		t.Fatalf("unexpected attempted actions %#v", executed)
	}
	if !lock.released {
		t.Fatal("failed synchronization did not release its lock")
	}

	if len(journalStore.savedJournals) != 3 {
		t.Fatalf("expected initial, completed, and failed journals, got %d", len(journalStore.savedJournals))
	}
	failed := journalStore.savedJournals[2]
	if failed.Status != state.JournalFailed ||
		failed.Actions[0].Status != state.ActionCompleted ||
		len(failed.Actions[0].Outputs) != 1 ||
		failed.Actions[0].Outputs[0].Name != "provider_state" ||
		failed.Actions[0].Outputs[0].Value != "running" ||
		failed.Actions[1].Status != state.ActionFailed ||
		failed.Actions[1].Failure != actionFailure.Error() ||
		failed.Actions[2].Status != state.ActionPending {
		t.Fatalf("unexpected failed journal %#v", failed)
	}

	details := make(map[string]string, len(commandError.Details))
	for _, detail := range commandError.Details {
		details[detail.Name] = detail.Value
	}
	if details["completed_actions"] != "01-prepare-provider" ||
		details["failed_action"] != "02-prepare-runtime" ||
		details["not_attempted_actions"] != "03-install-dependencies" ||
		details["observed_partial_state"] !=
			"01-prepare-provider:provider_state=running" ||
		details["safe_retry_command"] !=
			"elefante sync --approve-plan sha256:approved" ||
		details["manual_recovery"] == "" {
		t.Fatalf("unexpected synchronization failure details %#v", commandError.Details)
	}
}

func TestSyncEngineRetriesFromMatchingFailedJournal(t *testing.T) {
	actionFailure := errors.New("temporary provider failure")
	journalStore := &fakeSyncState{}
	var failSecond = true
	var executed []string
	engine := app.NewSyncEngine(app.SyncEngineDependencies{
		State: journalStore,
		AcquireLock: func(string) (app.SyncLock, error) {
			return &fakeSyncLock{acquired: true}, nil
		},
		ExecuteAction: func(
			_ context.Context,
			execution app.SyncActionExecution,
		) (app.SyncActionResult, error) {
			executed = append(executed, execution.Action.ID)
			if failSecond && execution.Action.ID == "02-runtime" {
				return app.SyncActionResult{}, actionFailure
			}

			return app.SyncActionResult{}, nil
		},
	})
	analysis := app.Analysis{
		Facts: model.ProjectFacts{
			Identity: model.ProjectIdentity{
				IdentityKey: "sha256:project",
			},
		},
		Plan: model.Plan{
			Digest: "sha256:approved",
			Actions: []model.PlanAction{
				{ID: "01-provider", Kind: model.ActionPrepareProvider},
				{ID: "02-runtime", Kind: model.ActionPrepareRuntime},
				{ID: "03-install", Kind: model.ActionInstallDependencies},
			},
		},
	}

	_, err := engine.Apply(t.Context(), app.SyncExecution{Analysis: analysis})
	if !errors.Is(err, actionFailure) {
		t.Fatalf("seed failed journal: %v", err)
	}
	failSecond = false
	executed = nil
	journalStore.savedJournals = nil

	result, err := engine.Apply(t.Context(), app.SyncExecution{Analysis: analysis})
	if err != nil {
		t.Fatalf("retry matching failed journal: %v", err)
	}
	if !slices.Equal(executed, []string{"02-runtime", "03-install"}) {
		t.Fatalf("retry did not resume from failed action: %#v", executed)
	}
	if !slices.Equal(
		result.CompletedActions,
		[]string{"01-provider", "02-runtime", "03-install"},
	) {
		t.Fatalf("unexpected retry result %#v", result)
	}
	if len(journalStore.savedJournals) != 4 {
		t.Fatalf("expected retry checkpoint, two completions, and success, got %d", len(journalStore.savedJournals))
	}
	retryCheckpoint := journalStore.savedJournals[0]
	if retryCheckpoint.Status != state.JournalPending ||
		retryCheckpoint.Actions[0].Status != state.ActionCompleted ||
		retryCheckpoint.Actions[1].Status != state.ActionPending ||
		retryCheckpoint.Actions[1].Failure != "" ||
		retryCheckpoint.Actions[2].Status != state.ActionPending {
		t.Fatalf("unexpected retry checkpoint %#v", retryCheckpoint)
	}
	final := journalStore.savedJournals[len(journalStore.savedJournals)-1]
	if final.Status != state.JournalSucceeded {
		t.Fatalf("retry did not finalize journal %#v", final)
	}
}

func TestSyncEngineRunsOnlyProviderProvenSafeCompensation(t *testing.T) {
	actionFailure := errors.New("final action failed")
	journalStore := &fakeSyncState{}
	var compensated []string
	engine := app.NewSyncEngine(app.SyncEngineDependencies{
		State: journalStore,
		AcquireLock: func(string) (app.SyncLock, error) {
			return &fakeSyncLock{acquired: true}, nil
		},
		ExecuteAction: func(
			_ context.Context,
			execution app.SyncActionExecution,
		) (app.SyncActionResult, error) {
			switch execution.Action.ID {
			case "01-safe":
				return app.SyncActionResult{
					Compensation: &app.SyncCompensation{
						Safe: true,
						Action: model.PlanAction{
							ID:   "compensate-01-safe",
							Kind: model.ActionPrepareProvider,
						},
					},
				}, nil
			case "02-unsafe":
				return app.SyncActionResult{
					Compensation: &app.SyncCompensation{
						Safe: false,
						Action: model.PlanAction{
							ID:   "compensate-02-unsafe",
							Kind: model.ActionPrepareRuntime,
						},
					},
				}, nil
			default:
				return app.SyncActionResult{}, actionFailure
			}
		},
		CompensateAction: func(
			_ context.Context,
			execution app.SyncCompensationExecution,
		) error {
			compensated = append(compensated, execution.Compensation.Action.ID)

			return nil
		},
	})
	analysis := app.Analysis{
		Facts: model.ProjectFacts{
			Identity: model.ProjectIdentity{
				IdentityKey: "sha256:project",
			},
		},
		Plan: model.Plan{
			Digest: "sha256:approved",
			Actions: []model.PlanAction{
				{
					ID:         "01-safe",
					Kind:       model.ActionPrepareProvider,
					Reversible: true,
				},
				{
					ID:         "02-unsafe",
					Kind:       model.ActionPrepareRuntime,
					Reversible: true,
				},
				{
					ID:   "03-fails",
					Kind: model.ActionInstallDependencies,
				},
			},
		},
	}

	_, err := engine.Apply(t.Context(), app.SyncExecution{Analysis: analysis})
	var commandError *model.Error
	if !errors.As(err, &commandError) ||
		commandError.Code != model.ErrorSync {
		t.Fatalf("expected synchronization failure, got %v", err)
	}
	if !slices.Equal(compensated, []string{"compensate-01-safe"}) {
		t.Fatalf("unexpected compensation actions %#v", compensated)
	}
	finalJournal := journalStore.savedJournals[len(journalStore.savedJournals)-1]
	if !finalJournal.Actions[0].Compensated ||
		finalJournal.Actions[1].Compensated {
		t.Fatalf("unexpected compensation journal %#v", finalJournal)
	}
	details := make(map[string]string, len(commandError.Details))
	for _, detail := range commandError.Details {
		details[detail.Name] = detail.Value
	}
	if details["compensated_actions"] != "01-safe" ||
		details["manual_recovery_actions"] != "02-unsafe" {
		t.Fatalf("unexpected compensation report %#v", commandError.Details)
	}
}

func TestSyncEngineRecordsSuccessfulEnvironmentAsFinalAction(t *testing.T) {
	syncState := &fakeSyncState{}
	var executed []string
	engine := app.NewSyncEngine(app.SyncEngineDependencies{
		State: syncState,
		AcquireLock: func(string) (app.SyncLock, error) {
			return &fakeSyncLock{acquired: true}, nil
		},
		ExecuteAction: func(
			_ context.Context,
			execution app.SyncActionExecution,
		) (app.SyncActionResult, error) {
			executed = append(executed, execution.Action.ID)

			return app.SyncActionResult{}, nil
		},
	})
	analysis := app.Analysis{
		Facts: model.ProjectFacts{
			Identity: model.ProjectIdentity{
				IdentityKey: "sha256:project",
			},
		},
		Plan: model.Plan{
			Digest: "sha256:approved",
			Provider: model.ProviderSelection{
				Name: "native",
			},
			Inputs: []model.InputFingerprint{
				{
					Path:   "/workspace/composer.json",
					Kind:   "composer_manifest",
					SHA256: "manifest",
				},
			},
			Actions: []model.PlanAction{
				{ID: "01-install", Kind: model.ActionInstallDependencies},
				{ID: "02-record", Kind: model.ActionRecordState},
			},
		},
	}

	result, err := engine.Apply(t.Context(), app.SyncExecution{
		Analysis: analysis,
	})
	if err != nil {
		t.Fatalf("record synchronized environment: %v", err)
	}
	if !slices.Equal(executed, []string{"01-install"}) {
		t.Fatalf("record state leaked to provider executor %#v", executed)
	}
	if len(syncState.savedEnvironments) != 1 {
		t.Fatalf("expected one environment record, got %#v", syncState.savedEnvironments)
	}
	record := syncState.savedEnvironments[0]
	if record.ProjectIdentity != "sha256:project" ||
		record.PlanDigest != "sha256:approved" ||
		record.Provider.Name != "native" ||
		!slices.Equal(
			record.CompletedActions,
			[]string{"01-install", "02-record"},
		) {
		t.Fatalf("unexpected environment record %#v", record)
	}
	if !slices.Equal(
		result.CompletedActions,
		[]string{"01-install", "02-record"},
	) {
		t.Fatalf("unexpected synchronization result %#v", result)
	}
}

func TestSyncEngineRequiresAndPersistsExactComposerTrust(t *testing.T) {
	syncState := &fakeSyncState{}
	var executionCalls int
	engine := app.NewSyncEngine(app.SyncEngineDependencies{
		State: syncState,
		AcquireLock: func(string) (app.SyncLock, error) {
			return &fakeSyncLock{acquired: true}, nil
		},
		ExecuteAction: func(
			context.Context,
			app.SyncActionExecution,
		) (app.SyncActionResult, error) {
			executionCalls++

			return app.SyncActionResult{}, nil
		},
	})
	requirement := model.TrustRequirement{
		Class:       model.TrustComposerScripts,
		Fingerprint: "sha256:trust-fingerprint",
	}
	analysis := app.Analysis{
		Facts: model.ProjectFacts{
			Identity: model.ProjectIdentity{
				IdentityKey: "sha256:project",
			},
		},
		Plan: model.Plan{
			Digest: "sha256:approved",
			Trust:  []model.TrustRequirement{requirement},
			Actions: []model.PlanAction{
				{
					ID:    "01-install",
					Kind:  model.ActionInstallDependencies,
					Trust: model.TrustComposerScripts,
				},
			},
		},
	}

	_, err := engine.Apply(t.Context(), app.SyncExecution{
		Analysis: analysis,
	})
	var commandError *model.Error
	if !errors.As(err, &commandError) ||
		commandError.Code != model.ErrorTrust {
		t.Fatalf("expected missing trust approval error, got %v", err)
	}
	if executionCalls != 0 ||
		len(syncState.savedJournals) != 0 ||
		len(syncState.savedTrusts) != 0 {
		t.Fatalf("missing trust reached mutation boundary %#v", syncState)
	}

	result, err := engine.Apply(t.Context(), app.SyncExecution{
		Analysis:      analysis,
		TrustApproved: true,
	})
	if err != nil {
		t.Fatalf("apply approved Composer trust: %v", err)
	}
	if !slices.Equal(result.CompletedActions, []string{"01-install"}) ||
		executionCalls != 1 {
		t.Fatalf("trusted action did not execute once, result %#v", result)
	}
	if len(syncState.savedTrusts) != 1 ||
		len(syncState.savedTrusts[0].Approvals) != 1 ||
		syncState.savedTrusts[0].Approvals[0].Class !=
			model.TrustComposerScripts ||
		syncState.savedTrusts[0].Approvals[0].Fingerprint !=
			requirement.Fingerprint {
		t.Fatalf("unexpected persisted trust %#v", syncState.savedTrusts)
	}
	if len(syncState.mutationOrder) < 2 ||
		syncState.mutationOrder[0] != "journal" ||
		syncState.mutationOrder[1] != "trust" {
		t.Fatalf("trust was not journaled before persistence %#v", syncState.mutationOrder)
	}
}

func TestSyncActionServiceInvokesOfficialComposerWithArgumentVectors(t *testing.T) {
	provider := &fakeProvider{name: "native"}
	runner := &fakeSyncRunner{}
	service := app.NewSyncActionService(app.SyncActionServiceDependencies{
		Providers: []providers.Provider{provider},
		Runner:    runner,
	})
	install := model.PlanAction{
		ID:   "01-install",
		Kind: model.ActionInstallDependencies,
		Inputs: []model.ActionInput{
			{Name: "composer", Value: "sha256:composer"},
			{Name: "provider", Value: "native"},
			{Name: "working_directory", Value: "/workspace"},
		},
		ExpectedOutputs: []model.ActionOutput{
			{Name: "dependencies", Value: "installed"},
		},
	}
	verify := model.PlanAction{
		ID:   "02-verify",
		Kind: model.ActionVerifyPlatform,
		Inputs: []model.ActionInput{
			{Name: "provider", Value: "native"},
			{Name: "working_directory", Value: "/workspace"},
		},
		ExpectedOutputs: []model.ActionOutput{
			{Name: "platform", Value: "verified"},
		},
	}
	analysis := app.Analysis{
		Observations: []model.ProviderObservation{
			{
				Provider: "native",
				Composer: []model.ComposerObservation{
					{
						Version:  "2.9.5",
						Source:   "provider",
						Path:     "/provider/composer",
						Identity: "sha256:composer",
					},
				},
			},
		},
		Plan: model.Plan{
			Provider: model.ProviderSelection{Name: "native"},
			Actions:  []model.PlanAction{install, verify},
		},
	}

	installResult, err := service.Execute(
		t.Context(),
		app.SyncActionExecution{
			Analysis:       analysis,
			Action:         install,
			NonInteractive: true,
		},
	)
	if err != nil {
		t.Fatalf("execute Composer install action: %v", err)
	}
	verifyResult, err := service.Execute(
		t.Context(),
		app.SyncActionExecution{
			Analysis:       analysis,
			Action:         verify,
			NonInteractive: true,
		},
	)
	if err != nil {
		t.Fatalf("execute Composer platform verification: %v", err)
	}
	if !slices.Equal(installResult.Outputs, install.ExpectedOutputs) ||
		!slices.Equal(verifyResult.Outputs, verify.ExpectedOutputs) {
		t.Fatalf(
			"unexpected Composer action outputs\ninstall: %#v\nverify: %#v",
			installResult,
			verifyResult,
		)
	}

	if len(provider.executionRequests) != 2 {
		t.Fatalf("expected two provider execution requests, got %#v", provider.executionRequests)
	}
	expectedRequests := []providers.ExecutionRequest{
		{
			Executable:       "/provider/composer",
			Arguments:        []string{"install", "--no-interaction"},
			WorkingDirectory: "/workspace",
		},
		{
			Executable:       "/provider/composer",
			Arguments:        []string{"check-platform-reqs"},
			WorkingDirectory: "/workspace",
		},
	}
	for index, expected := range expectedRequests {
		actual := provider.executionRequests[index]
		if actual.Executable != expected.Executable ||
			actual.WorkingDirectory != expected.WorkingDirectory ||
			!slices.Equal(actual.Arguments, expected.Arguments) {
			t.Fatalf(
				"execution request %d lost argument semantics\nactual:   %#v\nexpected: %#v",
				index,
				actual,
				expected,
			)
		}
	}
	if len(runner.commands) != 2 {
		t.Fatalf("expected two exact runner commands, got %#v", runner.commands)
	}
	for index, command := range runner.commands {
		expected := expectedRequests[index]
		if command.Executable != expected.Executable ||
			command.WorkingDirectory != expected.WorkingDirectory ||
			!slices.Equal(command.Arguments, expected.Arguments) {
			t.Fatalf("runner command %d changed provider specification %#v", index, command)
		}
	}
}

func TestSyncActionServiceMapsProviderProvenCompensation(t *testing.T) {
	provider := &fakeProvider{
		name: "ddev",
		applyResult: providers.ActionResult{
			Outputs: []model.ActionOutput{
				{Name: "provider_state", Value: "running"},
			},
			Compensation: &providers.ActionCompensation{
				Safe: true,
				Action: model.PlanAction{
					ID:   "stop-provider",
					Kind: model.ActionPrepareProvider,
				},
			},
		},
	}
	service := app.NewSyncActionService(app.SyncActionServiceDependencies{
		Providers: []providers.Provider{provider},
		Runner:    &fakeSyncRunner{},
	})
	action := model.PlanAction{
		ID:         "start-provider",
		Kind:       model.ActionPrepareProvider,
		Reversible: true,
	}
	result, err := service.Execute(t.Context(), app.SyncActionExecution{
		Analysis: app.Analysis{
			Plan: model.Plan{
				Provider: model.ProviderSelection{Name: "ddev"},
			},
		},
		Action: action,
	})
	if err != nil {
		t.Fatalf("apply provider action: %v", err)
	}
	if len(provider.applyRequests) != 1 ||
		provider.applyRequests[0].Action.ID != action.ID {
		t.Fatalf("unexpected provider apply requests %#v", provider.applyRequests)
	}
	if len(result.Outputs) != 1 ||
		result.Outputs[0].Value != "running" ||
		result.Compensation == nil ||
		!result.Compensation.Safe ||
		result.Compensation.Action.ID != "stop-provider" {
		t.Fatalf("provider compensation was not preserved %#v", result)
	}
	if err := service.Compensate(
		t.Context(),
		app.SyncCompensationExecution{
			Analysis: app.Analysis{
				Plan: model.Plan{
					Provider: model.ProviderSelection{Name: "ddev"},
				},
			},
			OriginalAction: action,
			Compensation:   *result.Compensation,
		},
	); err != nil {
		t.Fatalf("apply proven safe provider compensation: %v", err)
	}
	if len(provider.applyRequests) != 2 ||
		provider.applyRequests[1].Action.ID != "stop-provider" {
		t.Fatalf("unexpected provider compensation requests %#v", provider.applyRequests)
	}
}

func TestSyncActionServiceReverifiesManagedComposerBeforeExecution(t *testing.T) {
	provider := &fakeProvider{name: "native"}
	runner := &fakeSyncRunner{}
	var acquireRequests []composer.AcquireRequest
	service := app.NewSyncActionService(app.SyncActionServiceDependencies{
		Providers: []providers.Provider{provider},
		Runner:    runner,
		AcquireComposer: func(
			_ context.Context,
			request composer.AcquireRequest,
		) (composer.Executable, error) {
			acquireRequests = append(acquireRequests, request)

			return composer.Executable{
				Version:  request.Release.Version,
				Source:   composer.SourceManaged,
				Path:     "/verified/cache/composer.phar",
				Identity: request.Release.SHA256,
				SHA256:   request.Release.SHA256,
			}, nil
		},
	})
	prepare := model.PlanAction{
		ID:   "01-prepare-composer",
		Kind: model.ActionPrepareComposer,
		Inputs: []model.ActionInput{
			{Name: "version", Value: "2.9.5"},
			{
				Name:  "url",
				Value: "https://getcomposer.example/download/2.9.5/composer.phar",
			},
			{Name: "sha256", Value: "sha256:composer"},
			{
				Name:  "metadata_url",
				Value: "https://getcomposer.example/versions",
			},
		},
		ExpectedOutputs: []model.ActionOutput{
			{Name: "composer", Value: "sha256:composer"},
		},
	}
	install := model.PlanAction{
		ID:   "02-install",
		Kind: model.ActionInstallDependencies,
		Inputs: []model.ActionInput{
			{Name: "composer", Value: "sha256:composer"},
			{Name: "provider", Value: "native"},
			{Name: "working_directory", Value: "/workspace"},
		},
	}
	analysis := app.Analysis{
		Observations: []model.ProviderObservation{
			{
				Provider: "native",
				Composer: []model.ComposerObservation{
					{
						Version:  "2.9.5",
						Source:   composer.SourceManaged,
						Path:     "/planned/cache/composer.phar",
						Identity: "sha256:composer",
					},
				},
			},
		},
		Plan: model.Plan{
			Provider: model.ProviderSelection{Name: "native"},
			Actions:  []model.PlanAction{prepare, install},
		},
	}

	if _, err := service.Execute(
		t.Context(),
		app.SyncActionExecution{Analysis: analysis, Action: prepare},
	); err != nil {
		t.Fatalf("prepare managed Composer: %v", err)
	}
	if _, err := service.Execute(
		t.Context(),
		app.SyncActionExecution{Analysis: analysis, Action: install},
	); err != nil {
		t.Fatalf("execute reverified managed Composer: %v", err)
	}
	if len(acquireRequests) != 2 ||
		acquireRequests[0].Offline ||
		!acquireRequests[1].Offline ||
		acquireRequests[0].Release.SHA256 != "sha256:composer" ||
		acquireRequests[1].Release != acquireRequests[0].Release {
		t.Fatalf("unexpected managed Composer verification requests %#v", acquireRequests)
	}
	if len(runner.commands) != 1 ||
		runner.commands[0].Executable != "/verified/cache/composer.phar" ||
		!slices.Equal(runner.commands[0].Arguments, []string{"install"}) {
		t.Fatalf("unverified managed Composer reached execution %#v", runner.commands)
	}
}

type fakeSyncState struct {
	journal           state.ActionJournal
	journalFound      bool
	savedJournals     []state.ActionJournal
	savedEnvironments []state.EnvironmentRecord
	trust             state.TrustRecord
	trustFound        bool
	savedTrusts       []state.TrustRecord
	mutationOrder     []string
}

func (store *fakeSyncState) SaveJournal(
	_ string,
	journal state.ActionJournal,
) error {
	cloned := journal
	cloned.Actions = append([]state.JournalAction(nil), journal.Actions...)
	store.journal = cloned
	store.journalFound = true
	store.savedJournals = append(store.savedJournals, cloned)
	store.mutationOrder = append(store.mutationOrder, "journal")

	return nil
}

func (store *fakeSyncState) LoadJournal(
	_ string,
) (state.ActionJournal, bool, error) {
	return store.journal, store.journalFound, nil
}

func (store *fakeSyncState) SaveEnvironment(
	_ string,
	environment state.EnvironmentRecord,
) error {
	store.savedEnvironments = append(store.savedEnvironments, environment)
	store.mutationOrder = append(store.mutationOrder, "environment")

	return nil
}

func (store *fakeSyncState) LoadTrust(
	string,
) (state.TrustRecord, bool, error) {
	return store.trust, store.trustFound, nil
}

func (store *fakeSyncState) SaveTrust(
	_ string,
	trust state.TrustRecord,
) error {
	store.trust = trust
	store.trustFound = true
	store.savedTrusts = append(store.savedTrusts, trust)
	store.mutationOrder = append(store.mutationOrder, "trust")

	return nil
}

type fakeSyncLock struct {
	acquired bool
	released bool
}

type fakeSyncRunner struct {
	commands []executor.Command
}

func (runner *fakeSyncRunner) LookPath(string) (string, error) {
	return "", errors.New("unexpected executable lookup")
}

func (runner *fakeSyncRunner) Output(
	_ context.Context,
	command executor.Command,
) ([]byte, error) {
	runner.commands = append(runner.commands, command)

	return []byte("ok"), nil
}

func (lock *fakeSyncLock) Release() error {
	lock.released = true

	return nil
}
