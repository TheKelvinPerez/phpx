package ddev_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/elefantephp/elefante/internal/executor"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
	"github.com/elefantephp/elefante/internal/providers/ddev"
	"github.com/elefantephp/elefante/internal/providers/providertest"
)

func TestInspectReportsUnavailableWhenDDEVIsMissing(t *testing.T) {
	runner := &fakeRunner{}
	provider := ddev.New(ddev.Dependencies{
		Runner: runner,
		ConfigExists: func(string) (bool, error) {
			return false, nil
		},
	})

	observation, err := provider.Inspect(
		t.Context(),
		providers.InspectRequest{
			Project: model.ProjectIdentity{
				ComposerRoot: "/workspace",
			},
			Offline: true,
		},
	)
	if err != nil {
		t.Fatalf("inspect missing DDEV: %v", err)
	}

	if observation.Provider != "ddev" || observation.Available {
		t.Fatalf("expected unavailable DDEV observation, got %#v", observation)
	}
	diagnostic := diagnosticByCode(
		t,
		observation.Diagnostics,
		"ELEFANTE_DDEV_NOT_FOUND",
	)
	if diagnostic.Severity != model.SeverityWarning ||
		diagnostic.Provider != "ddev" {
		t.Fatalf("unexpected missing DDEV diagnostic %#v", diagnostic)
	}
	if len(observation.Capabilities) != 0 {
		t.Fatalf("missing DDEV cannot expose capabilities: %#v", observation)
	}
	if len(observation.Fingerprint) != 71 ||
		observation.Fingerprint[:7] != "sha256:" {
		t.Fatalf("expected stable observation fingerprint, got %q", observation.Fingerprint)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("missing DDEV must not run commands, got %#v", runner.commands)
	}
}

func TestInspectReportsEngineAndMissingProjectConfiguration(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"ddev": "/fixture/bin/ddev",
		},
		outputs: map[string][]byte{
			"/fixture/bin/ddev\x00version\x00--json-output": readFixture(
				t,
				"version.json",
			),
		},
	}
	provider := ddev.New(ddev.Dependencies{
		Runner: runner,
		ConfigExists: func(path string) (bool, error) {
			if path != "/workspace/.ddev/config.yaml" {
				t.Fatalf("unexpected DDEV config path %q", path)
			}

			return false, nil
		},
	})

	observation, err := provider.Inspect(
		t.Context(),
		providers.InspectRequest{
			Project: model.ProjectIdentity{
				ComposerRoot: "/workspace",
			},
			Offline: true,
		},
	)
	if err != nil {
		t.Fatalf("inspect unconfigured DDEV: %v", err)
	}

	if !observation.Available ||
		observation.Version != "1.24.8" ||
		observation.Platform != "darwin" ||
		observation.Architecture != "arm64" ||
		observation.State != model.ProviderStateUnconfigured {
		t.Fatalf("unexpected DDEV identity %#v", observation)
	}
	if len(observation.Engines) != 1 ||
		observation.Engines[0].Name != "docker" ||
		observation.Engines[0].Version != "29.4.0" ||
		observation.Engines[0].Platform != "orbstack" ||
		observation.Engines[0].Source.Path != "/fixture/bin/ddev" {
		t.Fatalf("unexpected DDEV engine %#v", observation.Engines)
	}
	diagnostic := diagnosticByCode(
		t,
		observation.Diagnostics,
		"ELEFANTE_DDEV_CONFIG_MISSING",
	)
	if len(diagnostic.Sources) != 1 ||
		diagnostic.Sources[0].Path != "/workspace/.ddev/config.yaml" {
		t.Fatalf("unexpected missing config diagnostic %#v", diagnostic)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("missing config must stop after version inspection: %#v", runner.commands)
	}
	if len(runner.commands[0].Environment) != 1 ||
		runner.commands[0].Environment[0] != "DDEV_NO_INSTRUMENTATION=true" {
		t.Fatalf("doctor must disable DDEV instrumentation: %#v", runner.commands[0])
	}
}

func TestInspectFingerprintIncludesMissingConfigurationProvenance(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"ddev": "/fixture/bin/ddev",
		},
		outputs: map[string][]byte{
			"/fixture/bin/ddev\x00version\x00--json-output": readFixture(
				t,
				"version.json",
			),
		},
	}
	provider := ddev.New(ddev.Dependencies{
		Runner: runner,
		ConfigExists: func(string) (bool, error) {
			return false, nil
		},
	})

	first, err := provider.Inspect(t.Context(), providers.InspectRequest{
		Project: model.ProjectIdentity{
			ComposerRoot: "/workspace/first",
		},
	})
	if err != nil {
		t.Fatalf("inspect first unconfigured project: %v", err)
	}
	second, err := provider.Inspect(t.Context(), providers.InspectRequest{
		Project: model.ProjectIdentity{
			ComposerRoot: "/workspace/second",
		},
	})
	if err != nil {
		t.Fatalf("inspect second unconfigured project: %v", err)
	}

	if first.Fingerprint == second.Fingerprint {
		t.Fatalf(
			"missing configuration provenance must alter fingerprint %q",
			first.Fingerprint,
		)
	}
}

func TestInspectReportsStoppedConfiguredProject(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"ddev": "/fixture/bin/ddev",
		},
		outputs: map[string][]byte{
			"/fixture/bin/ddev\x00version\x00--json-output": readFixture(
				t,
				"version.json",
			),
			"/fixture/bin/ddev\x00describe\x00--json-output\x00--skip-hooks": readFixture(
				t,
				"describe-stopped.json",
			),
		},
	}
	provider := ddev.New(ddev.Dependencies{
		Runner: runner,
		ConfigExists: func(string) (bool, error) {
			return true, nil
		},
	})

	observation, err := provider.Inspect(
		t.Context(),
		providers.InspectRequest{
			Project: model.ProjectIdentity{
				ComposerRoot: "/workspace",
			},
			Offline: true,
		},
	)
	if err != nil {
		t.Fatalf("inspect stopped DDEV project: %v", err)
	}

	if observation.State != model.ProviderStateStopped {
		t.Fatalf("expected stopped DDEV state, got %#v", observation)
	}
	if len(observation.Runtimes) != 1 ||
		observation.Runtimes[0].Name != "php" ||
		observation.Runtimes[0].Version != "8.3" ||
		observation.Runtimes[0].Source.Path != "/workspace/.ddev/config.yaml" ||
		observation.Runtimes[0].Source.Field != "php_version" {
		t.Fatalf("unexpected configured PHP observation %#v", observation.Runtimes)
	}
	if len(observation.Extensions) != 0 ||
		len(observation.Composer) != 0 {
		t.Fatalf("stopped project must not claim live container state: %#v", observation)
	}
	diagnostic := diagnosticByCode(
		t,
		observation.Diagnostics,
		"ELEFANTE_DDEV_STOPPED",
	)
	if diagnostic.Severity != model.SeverityWarning ||
		diagnostic.Provider != "ddev" {
		t.Fatalf("unexpected stopped diagnostic %#v", diagnostic)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("stopped inspection must not use ddev exec: %#v", runner.commands)
	}
	describeCommand := runner.commands[1]
	if describeCommand.WorkingDirectory != "/workspace" {
		t.Fatalf("DDEV describe must run from project root: %#v", describeCommand)
	}
}

func TestInspectReportsRunningPHPComposerAndExtensions(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"ddev": "/fixture/bin/ddev",
		},
		outputs: map[string][]byte{
			"/fixture/bin/ddev\x00version\x00--json-output": readFixture(
				t,
				"version.json",
			),
			"/fixture/bin/ddev\x00describe\x00--json-output\x00--skip-hooks": readFixture(
				t,
				"describe-running.json",
			),
		},
		execOutputs: map[string][]byte{
			"php":      readFixture(t, "php-inspection.json"),
			"which":    []byte("/usr/local/bin/composer\n"),
			"composer": []byte("Composer version 2.8.12 2026-01-10 10:00:00\n"),
		},
	}
	provider := ddev.New(ddev.Dependencies{
		Runner: runner,
		ConfigExists: func(string) (bool, error) {
			return true, nil
		},
	})

	observation, err := provider.Inspect(
		t.Context(),
		providers.InspectRequest{
			Project: model.ProjectIdentity{
				ComposerRoot: "/workspace",
			},
			Offline: true,
		},
	)
	if err != nil {
		t.Fatalf("inspect running DDEV project: %v", err)
	}

	if observation.State != model.ProviderStateRunning ||
		len(observation.Diagnostics) != 0 {
		t.Fatalf("unexpected running state %#v", observation)
	}
	if len(observation.Runtimes) != 1 ||
		observation.Runtimes[0].Version != "8.3.28" ||
		observation.Runtimes[0].SAPI != "cli" ||
		observation.Runtimes[0].Source.Path != "/usr/bin/php" ||
		observation.Runtimes[0].Source.Field != "elefante-phase9:web" {
		t.Fatalf("unexpected live PHP observation %#v", observation.Runtimes)
	}
	expectedExtensions := []string{
		"ext-curl",
		"ext-json",
		"ext-zend-opcache",
	}
	if len(observation.Extensions) != len(expectedExtensions) {
		t.Fatalf("unexpected live extensions %#v", observation.Extensions)
	}
	for index, expected := range expectedExtensions {
		if observation.Extensions[index].Name != expected ||
			observation.Extensions[index].Source.Path != "/usr/bin/php" {
			t.Fatalf("unexpected extension %d: %#v", index, observation.Extensions[index])
		}
	}
	if len(observation.Composer) != 1 ||
		observation.Composer[0].Version != "2.8.12" ||
		observation.Composer[0].Path != "/usr/local/bin/composer" ||
		observation.Composer[0].Source != "ddev" ||
		observation.Composer[0].Reference.Field != "elefante-phase9:web" ||
		len(observation.Composer[0].Identity) != 71 {
		t.Fatalf("unexpected DDEV Composer observation %#v", observation.Composer)
	}
	if len(runner.commands) != 5 {
		t.Fatalf("expected version, describe, and three live probes: %#v", runner.commands)
	}
	phpCommand := runner.commands[2]
	separator := argumentIndex(phpCommand.Arguments, "--")
	if separator < 0 ||
		len(phpCommand.Arguments) != separator+4 ||
		phpCommand.Arguments[separator+1] != "php" ||
		phpCommand.Arguments[separator+2] != "-r" ||
		phpCommand.WorkingDirectory != "/workspace" {
		t.Fatalf("unexpected argument safe PHP probe %#v", phpCommand)
	}
}

func TestInspectPreservesDDEVVersionProcessFailure(t *testing.T) {
	processFailure := errors.New("DDEV version process failed")
	command := "/fixture/bin/ddev\x00version\x00--json-output"
	provider := ddev.New(ddev.Dependencies{
		Runner: &fakeRunner{
			paths: map[string]string{
				"ddev": "/fixture/bin/ddev",
			},
			errors: map[string]error{
				command: processFailure,
			},
		},
	})

	_, err := provider.Inspect(t.Context(), providers.InspectRequest{
		Project: model.ProjectIdentity{
			ComposerRoot: "/workspace",
		},
	})
	if err == nil {
		t.Fatal("expected DDEV version process failure")
	}
	var providerFailure *model.Error
	if !errors.As(err, &providerFailure) {
		t.Fatalf("expected typed provider error, got %T: %v", err, err)
	}
	if providerFailure.Code != model.ErrorProvider ||
		providerFailure.Provider != "ddev" ||
		!providerFailure.Retryable {
		t.Fatalf("unexpected DDEV provider error %#v", providerFailure)
	}
	if !errors.Is(err, processFailure) {
		t.Fatalf("expected process cause to be preserved, got %v", err)
	}
}

func TestInspectRejectsMalformedDDEVVersionOutput(t *testing.T) {
	provider := ddev.New(ddev.Dependencies{
		Runner: &fakeRunner{
			paths: map[string]string{
				"ddev": "/fixture/bin/ddev",
			},
			outputs: map[string][]byte{
				"/fixture/bin/ddev\x00version\x00--json-output": []byte("{"),
			},
		},
	})

	_, err := provider.Inspect(t.Context(), providers.InspectRequest{
		Project: model.ProjectIdentity{
			ComposerRoot: "/workspace",
		},
	})
	if err == nil {
		t.Fatal("expected malformed DDEV version output to fail")
	}
	var providerFailure *model.Error
	if !errors.As(err, &providerFailure) ||
		providerFailure.Code != model.ErrorProvider ||
		providerFailure.Provider != "ddev" {
		t.Fatalf("expected typed DDEV provider error, got %T: %v", err, err)
	}
}

func TestPlanMakesMissingConfigurationAndStartReviewable(t *testing.T) {
	provider := ddev.New(ddev.Dependencies{
		Runner: &fakeRunner{},
	})
	request := providers.ProviderPlanRequest{
		Facts: model.ProjectFacts{
			Identity: model.ProjectIdentity{
				ComposerRoot:    "/workspace",
				ApplicationRoot: "/workspace",
				WorkspaceRoot:   "/workspace",
			},
			Frameworks: []model.FrameworkFact{
				{
					Kind:    model.FrameworkLaravelApplication,
					Primary: true,
				},
			},
		},
		Observation: model.ProviderObservation{
			Provider:  "ddev",
			Available: true,
			State:     model.ProviderStateUnconfigured,
		},
		Policy: model.PlanPolicy{},
	}

	result, err := provider.Plan(t.Context(), request)
	if err != nil {
		t.Fatalf("plan unconfigured DDEV: %v", err)
	}

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected reviewable actions, got %#v", result.Diagnostics)
	}
	if len(result.Actions) != 2 {
		t.Fatalf("expected configure and start actions, got %#v", result.Actions)
	}
	configure := result.Actions[0]
	if configure.Kind != model.ActionPrepareProvider ||
		configure.Effect != model.EffectProjectMutation ||
		configure.Network != model.NetworkNone ||
		!configure.Reversible ||
		actionInput(configure, "operation") != "configure" ||
		actionInput(configure, "config_path") !=
			"/workspace/.ddev/config.yaml" ||
		actionInput(configure, "project_type") != "laravel" {
		t.Fatalf("unexpected DDEV configure action %#v", configure)
	}
	start := result.Actions[1]
	if start.Kind != model.ActionPrepareProvider ||
		start.Effect != model.EffectProviderMutation ||
		start.Network != model.NetworkRequired ||
		actionInput(start, "operation") != "start" ||
		actionInput(start, "project_root") != "/workspace" {
		t.Fatalf("unexpected DDEV start action %#v", start)
	}
}

func TestPlanBlocksMissingConfigurationInFrozenMode(t *testing.T) {
	provider := ddev.New(ddev.Dependencies{
		Runner: &fakeRunner{},
	})
	result, err := provider.Plan(t.Context(), providers.ProviderPlanRequest{
		Facts: model.ProjectFacts{
			Identity: model.ProjectIdentity{
				ComposerRoot: "/workspace",
			},
		},
		Observation: model.ProviderObservation{
			Provider:  "ddev",
			Available: true,
			State:     model.ProviderStateUnconfigured,
		},
		Policy: model.PlanPolicy{
			Frozen: true,
		},
	})
	if err != nil {
		t.Fatalf("plan frozen unconfigured DDEV: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Fatalf("frozen mode must not configure DDEV: %#v", result.Actions)
	}
	diagnostic := diagnosticByCode(
		t,
		result.Diagnostics,
		"ELEFANTE_DDEV_CONFIG_FROZEN",
	)
	if diagnostic.Severity != model.SeverityError ||
		diagnostic.Provider != "ddev" ||
		len(diagnostic.Sources) != 1 ||
		diagnostic.Sources[0].Path != "/workspace/.ddev/config.yaml" {
		t.Fatalf("unexpected frozen DDEV diagnostic %#v", diagnostic)
	}
}

func TestPlanUsesSupportedDDEVProjectTypeForBedrock(t *testing.T) {
	provider := ddev.New(ddev.Dependencies{
		Runner: &fakeRunner{},
	})
	result, err := provider.Plan(t.Context(), providers.ProviderPlanRequest{
		Facts: model.ProjectFacts{
			Identity: model.ProjectIdentity{
				ComposerRoot: "/workspace",
			},
			Frameworks: []model.FrameworkFact{
				{
					Kind:    model.FrameworkBedrockWordPress,
					Primary: true,
				},
			},
		},
		Observation: model.ProviderObservation{
			Provider:  "ddev",
			Available: true,
			State:     model.ProviderStateUnconfigured,
		},
	})
	if err != nil {
		t.Fatalf("plan Bedrock DDEV configuration: %v", err)
	}

	if len(result.Actions) == 0 ||
		actionInput(result.Actions[0], "project_type") != "wordpress" {
		t.Fatalf("expected supported DDEV WordPress type, got %#v", result.Actions)
	}
}

func TestPlanStartsOnlyStoppedProjects(t *testing.T) {
	provider := ddev.New(ddev.Dependencies{
		Runner: &fakeRunner{},
	})
	facts := model.ProjectFacts{
		Identity: model.ProjectIdentity{
			ComposerRoot: "/workspace",
		},
	}

	stopped, err := provider.Plan(t.Context(), providers.ProviderPlanRequest{
		Facts: facts,
		Observation: model.ProviderObservation{
			Provider:  "ddev",
			Available: true,
			State:     model.ProviderStateStopped,
		},
	})
	if err != nil {
		t.Fatalf("plan stopped DDEV: %v", err)
	}
	if len(stopped.Actions) != 1 ||
		actionInput(stopped.Actions[0], "operation") != "start" {
		t.Fatalf("stopped DDEV must plan only start: %#v", stopped.Actions)
	}

	running, err := provider.Plan(t.Context(), providers.ProviderPlanRequest{
		Facts: facts,
		Observation: model.ProviderObservation{
			Provider:  "ddev",
			Available: true,
			State:     model.ProviderStateRunning,
		},
	})
	if err != nil {
		t.Fatalf("plan running DDEV: %v", err)
	}
	if len(running.Actions) != 0 || len(running.Diagnostics) != 0 {
		t.Fatalf("running DDEV needs no provider preparation: %#v", running)
	}
}

func TestProviderConformance(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"ddev": "/fixture/bin/ddev",
		},
		outputs: map[string][]byte{
			"/fixture/bin/ddev\x00version\x00--json-output": readFixture(
				t,
				"version.json",
			),
			"/fixture/bin/ddev\x00describe\x00--json-output\x00--skip-hooks": readFixture(
				t,
				"describe-running.json",
			),
		},
		execOutputs: map[string][]byte{
			"php":      readFixture(t, "php-inspection.json"),
			"which":    []byte("/usr/local/bin/composer\n"),
			"composer": []byte("Composer version 2.8.12 2026-01-10 10:00:00\n"),
		},
	}

	providertest.Run(t, providertest.Suite{
		Provider: ddev.New(ddev.Dependencies{
			Runner: runner,
			ConfigExists: func(string) (bool, error) {
				return true, nil
			},
		}),
		InspectRequest: providers.InspectRequest{
			Project: model.ProjectIdentity{
				ComposerRoot: "/workspace",
			},
			Offline: true,
		},
		ExecutionRequest: providers.ExecutionRequest{
			Executable: "php",
			Arguments: []string{
				"-r",
				`echo "with spaces";`,
				"; rm -rf /",
				"$(whoami)",
			},
			WorkingDirectory: "/workspace",
			Environment:      []string{"APP_ENV=test"},
		},
		AssertExecutionArguments: func(
			t *testing.T,
			spec providers.ExecutionSpec,
			request providers.ExecutionRequest,
		) {
			t.Helper()

			separator := argumentIndex(spec.Arguments, "--")
			expectedPrefix := []string{
				"--skip-hooks",
				"exec",
				"--raw",
				"--",
			}
			expected := append(
				[]string{request.Executable},
				request.Arguments...,
			)
			if separator < 0 ||
				!reflect.DeepEqual(
					spec.Arguments[:separator+1],
					expectedPrefix,
				) ||
				!reflect.DeepEqual(spec.Arguments[separator+1:], expected) {
				t.Fatalf(
					"DDEV changed wrapped arguments\nexpected: %#v\ngot:      %#v",
					expected,
					spec.Arguments,
				)
			}
		},
	})
}

type fakeRunner struct {
	paths       map[string]string
	outputs     map[string][]byte
	execOutputs map[string][]byte
	errors      map[string]error
	commands    []executor.Command
}

func actionInput(action model.PlanAction, name string) string {
	for _, input := range action.Inputs {
		if input.Name == name {
			return input.Value
		}
	}

	return ""
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read DDEV fixture %s: %v", name, err)
	}

	return content
}

func (runner *fakeRunner) LookPath(file string) (string, error) {
	if path, found := runner.paths[file]; found {
		return path, nil
	}

	return "", exec.ErrNotFound
}

func (runner *fakeRunner) Output(
	_ context.Context,
	command executor.Command,
) ([]byte, error) {
	runner.commands = append(runner.commands, command)
	key := commandKey(command)
	if err := runner.errors[key]; err != nil {
		return runner.outputs[key], err
	}
	if output, found := runner.outputs[key]; found {
		return append([]byte(nil), output...), nil
	}
	if separator := argumentIndex(command.Arguments, "--"); separator >= 0 &&
		separator+1 < len(command.Arguments) {
		executable := command.Arguments[separator+1]
		if output, found := runner.execOutputs[executable]; found {
			return append([]byte(nil), output...), nil
		}
	}

	return nil, errors.New("unexpected command: " + key)
}

func argumentIndex(arguments []string, expected string) int {
	for index, argument := range arguments {
		if argument == expected {
			return index
		}
	}

	return -1
}

func commandKey(command executor.Command) string {
	key := command.Executable
	for _, argument := range command.Arguments {
		key += "\x00" + argument
	}

	return key
}

func diagnosticByCode(
	t *testing.T,
	diagnostics []model.Diagnostic,
	code string,
) model.Diagnostic {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return diagnostic
		}
	}
	t.Fatalf("expected diagnostic %q, got %#v", code, diagnostics)

	return model.Diagnostic{}
}
