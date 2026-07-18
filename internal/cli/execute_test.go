package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/cli"
	"github.com/elefantephp/elefante/internal/discovery"
	"github.com/elefantephp/elefante/internal/executor"
	"github.com/elefantephp/elefante/internal/model"
)

func TestJSONVersionOwnsStandardOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	application := app.New(app.Dependencies{
		Build: model.BuildInfo{Version: "dev"},
	})

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{Application: application},
		cli.Execution{
			Arguments: []string{"--json", "version"},
			Input:     strings.NewReader(""),
			Output:    &stdout,
			Error:     &stderr,
		},
	)

	if exitCode != 0 {
		t.Fatalf("expected exit zero, got %d\nstderr:\n%s", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got:\n%s", stderr.String())
	}

	expected := readEventGolden(t, "version-success.ndjson")
	if stdout.String() != expected {
		t.Fatalf("JSON command output does not match golden\nexpected:\n%s\ngot:\n%s", expected, stdout.String())
	}
}

func TestJSONUsageErrorOwnsStandardOutputAndExitCode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	application := app.New(app.Dependencies{
		Build: model.BuildInfo{Version: "dev"},
	})

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{Application: application},
		cli.Execution{
			Arguments: []string{"--json", "unknown"},
			Input:     strings.NewReader(""),
			Output:    &stdout,
			Error:     &stderr,
		},
	)

	if exitCode != 2 {
		t.Fatalf("expected usage exit 2, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected JSON mode stderr to be empty, got:\n%s", stderr.String())
	}

	expected := readEventGolden(t, "usage-error.ndjson")
	if stdout.String() != expected {
		t.Fatalf("JSON error output does not match golden\nexpected:\n%s\ngot:\n%s", expected, stdout.String())
	}
}

func TestExecuteRedactsSecretsDerivedFromEnvironment(t *testing.T) {
	t.Parallel()

	const secret = "unknown-secret-command"
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{
			Application: app.New(app.Dependencies{}),
		},
		cli.Execution{
			Arguments:   []string{"--json", secret},
			Environment: []string{"API_TOKEN=" + secret},
			Input:       strings.NewReader(""),
			Output:      &stdout,
			Error:       &stderr,
		},
	)

	if exitCode != 2 {
		t.Fatalf("expected usage exit 2, got %d", exitCode)
	}
	if strings.Contains(stdout.String(), secret) {
		t.Fatalf("captured output leaked an environment secret: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "[REDACTED]") {
		t.Fatalf("expected redaction marker, got %s", stdout.String())
	}
}

func TestJSONRunPreservesChildExitAfterPostStartExecutionFailure(
	t *testing.T,
) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	executionFailure := errors.New("synthetic post-start stream failure")
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			context.Context,
			discovery.Request,
		) (model.ProjectFacts, error) {
			return cliCompatibleFacts(), nil
		},
		Providers: providerSet(compatibleCLIProvider()),
		ExecuteProcess: func(
			context.Context,
			executor.Command,
			executor.Streams,
		) (executor.Result, error) {
			return executor.Result{
				Started:  true,
				ExitCode: 29,
			}, executionFailure
		},
	})

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{Application: application},
		cli.Execution{
			Arguments: []string{
				"--json",
				"--provider", "native",
				"run",
				"--",
				"child",
			},
			Input:  strings.NewReader(""),
			Output: &stdout,
			Error:  &stderr,
		},
	)
	if exitCode != 29 {
		t.Fatalf(
			"expected child exit 29, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	if stderr.Len() != 0 {
		t.Fatalf("machine mode wrote raw stderr:\n%s", stderr.String())
	}

	var events []struct {
		Type    model.EventType `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		var event struct {
			Type    model.EventType `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode run failure event: %v", err)
		}
		events = append(events, event)
	}
	if len(events) != 3 ||
		events[0].Type != model.EventStarted ||
		events[1].Type != model.EventError ||
		events[2].Type != model.EventCompleted {
		t.Fatalf("unexpected post-start failure events %#v", events)
	}
	var completed model.CompletedPayload
	if err := json.Unmarshal(events[2].Payload, &completed); err != nil {
		t.Fatalf("decode post-start completion: %v", err)
	}
	if completed.Exit.Origin != model.ExitOriginChild ||
		completed.Exit.Code != 29 {
		t.Fatalf("post-start failure lost child completion %#v", completed)
	}
}

func TestNonterminalSyncNeverPromptsBeforeApprovalFailure(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var mutationCalls int
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			context.Context,
			discovery.Request,
		) (model.ProjectFacts, error) {
			return cliCompatibleFacts(), nil
		},
		Providers: providerSet(compatibleCLIProvider()),
		ExecuteApprovedPlan: func(context.Context, app.Analysis) error {
			mutationCalls++

			return nil
		},
	})

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{Application: application},
		cli.Execution{
			Arguments: []string{
				"--project",
				"/workspace",
				"--provider",
				"native",
				"sync",
			},
			Input:  failOnRead{},
			Output: &stdout,
			Error:  &stderr,
		},
	)

	if exitCode != 6 {
		t.Fatalf(
			"expected approval exit 6, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	if mutationCalls != 0 {
		t.Fatalf("approval failure reached mutation boundary %d times", mutationCalls)
	}
	if !strings.Contains(stderr.String(), "requires explicit approval") {
		t.Fatalf("expected approval error, got:\n%s", stderr.String())
	}
}

func TestSyncApprovalFlagsReachMutationOnlyForCurrentPlan(t *testing.T) {
	t.Parallel()

	var mutationCalls int
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			context.Context,
			discovery.Request,
		) (model.ProjectFacts, error) {
			return cliCompatibleFacts(), nil
		},
		Providers: providerSet(compatibleCLIProvider()),
		ExecuteApprovedPlan: func(context.Context, app.Analysis) error {
			mutationCalls++

			return nil
		},
	})
	planned, err := application.Plan(t.Context(), app.PlanRequest{
		ProjectPath: "/workspace",
		Provider:    "native",
	})
	if err != nil {
		t.Fatalf("build reviewed plan: %v", err)
	}

	run := func(arguments ...string) (int, string, string) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := cli.Execute(
			context.Background(),
			cli.Dependencies{Application: application},
			cli.Execution{
				Arguments: arguments,
				Input:     failOnRead{},
				Output:    &stdout,
				Error:     &stderr,
			},
		)

		return exitCode, stdout.String(), stderr.String()
	}

	exitCode, _, stderr := run(
		"--project", "/workspace",
		"--provider", "native",
		"--yes",
		"sync",
	)
	if exitCode != 0 {
		t.Fatalf("--yes exit %d\nstderr:\n%s", exitCode, stderr)
	}
	if mutationCalls != 1 {
		t.Fatalf("expected --yes mutation call, got %d", mutationCalls)
	}

	exitCode, stdout, stderr := run(
		"--json",
		"--project", "/workspace",
		"--provider", "native",
		"--approve-plan", planned.Plan.Digest,
		"sync",
	)
	if exitCode != 0 {
		t.Fatalf(
			"exact approval exit %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	if mutationCalls != 2 {
		t.Fatalf("expected exact approval mutation call, got %d", mutationCalls)
	}
	events := decodeCLIEvents(t, stdout)
	for _, event := range events {
		if event.Command != "sync" {
			t.Fatalf("expected sync event command, got %#v", event)
		}
	}

	exitCode, _, _ = run(
		"--project", "/workspace",
		"--provider", "native",
		"--approve-plan", "sha256:stale",
		"sync",
	)
	if exitCode != 7 {
		t.Fatalf("expected mismatch exit 7, got %d", exitCode)
	}
	if mutationCalls != 2 {
		t.Fatalf("plan mismatch reached mutation boundary, got %d calls", mutationCalls)
	}

	exitCode, _, _ = run(
		"--project", "/workspace",
		"--provider", "native",
		"--yes",
		"--approve-plan", planned.Plan.Digest,
		"sync",
	)
	if exitCode != 2 {
		t.Fatalf("expected mutually exclusive flag exit 2, got %d", exitCode)
	}
	if mutationCalls != 2 {
		t.Fatalf("usage error reached mutation boundary, got %d calls", mutationCalls)
	}
}

func TestTerminalSyncConfirmationRevalidatesDisplayedPlan(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var mutationCalls int
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			context.Context,
			discovery.Request,
		) (model.ProjectFacts, error) {
			return cliCompatibleFacts(), nil
		},
		Providers: providerSet(compatibleCLIProvider()),
		ExecuteApprovedPlan: func(context.Context, app.Analysis) error {
			mutationCalls++

			return nil
		},
	})

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{Application: application},
		cli.Execution{
			Arguments: []string{
				"--project", "/workspace",
				"--provider", "native",
				"sync",
			},
			Input:           strings.NewReader("yes\n"),
			InputIsTerminal: true,
			Output:          &stdout,
			Error:           &stderr,
		},
	)

	if exitCode != 0 {
		t.Fatalf(
			"expected confirmed sync exit zero, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	if mutationCalls != 1 {
		t.Fatalf("expected one mutation call, got %d", mutationCalls)
	}
	if !strings.Contains(stderr.String(), "Apply this exact plan? [y/N]:") {
		t.Fatalf("expected terminal approval prompt, got:\n%s", stderr.String())
	}
}

func TestNonInteractiveFlagDisablesTerminalPrompt(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			context.Context,
			discovery.Request,
		) (model.ProjectFacts, error) {
			return cliCompatibleFacts(), nil
		},
		Providers: providerSet(compatibleCLIProvider()),
		ExecuteApprovedPlan: func(context.Context, app.Analysis) error {
			t.Fatal("noninteractive approval failure reached mutation")

			return nil
		},
	})

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{Application: application},
		cli.Execution{
			Arguments: []string{
				"--project", "/workspace",
				"--provider", "native",
				"--non-interactive",
				"sync",
			},
			Input:           failOnRead{},
			InputIsTerminal: true,
			Output:          &stdout,
			Error:           &stderr,
		},
	)

	if exitCode != 6 {
		t.Fatalf(
			"expected approval exit 6, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
}

type cliEvent struct {
	Command string          `json:"command"`
	Type    model.EventType `json:"type"`
}

func decodeCLIEvents(t *testing.T, content string) []cliEvent {
	t.Helper()

	var events []cliEvent
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		var event cliEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode CLI event: %v\nline: %s", err, line)
		}
		events = append(events, event)
	}

	return events
}

type failOnRead struct{}

func (failOnRead) Read([]byte) (int, error) {
	panic("nonterminal input must never be read for approval")
}

func compatibleCLIProvider() *testProvider {
	return &testProvider{
		observation: model.ProviderObservation{
			Provider:  "native",
			Available: true,
			Runtimes: []model.RuntimeObservation{
				{
					Name:    "php",
					Version: "8.5.0",
					SAPI:    "cli",
					Source: model.SourceReference{
						Path: "/fixture/bin/php",
						Kind: "provider_executable",
					},
				},
			},
			Composer: []model.ComposerObservation{
				{
					Version:  "2.9.5",
					Source:   "system",
					Path:     "/fixture/bin/composer",
					Identity: "sha256:composer",
				},
			},
			Fingerprint: "sha256:native",
		},
	}
}

func TestJSONFlagAfterCommandSeparatorDoesNotChangeElefanteOutputMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	application := app.New(app.Dependencies{
		Build: model.BuildInfo{Version: "dev"},
	})

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{Application: application},
		cli.Execution{
			Arguments: []string{"version", "--", "--json"},
			Input:     strings.NewReader(""),
			Output:    &stdout,
			Error:     &stderr,
		},
	)

	if exitCode != 2 {
		t.Fatalf("expected usage exit 2, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected human mode stdout to be empty, got:\n%s", stdout.String())
	}
	if !strings.HasPrefix(stderr.String(), "Error: unknown command") {
		t.Fatalf("expected a human usage error, got:\n%s", stderr.String())
	}
}

func TestJSONHelpContainsOnlyProtocolEvents(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	application := app.New(app.Dependencies{
		Build: model.BuildInfo{Version: "dev"},
	})

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{Application: application},
		cli.Execution{
			Arguments: []string{"--json", "--help"},
			Input:     strings.NewReader(""),
			Output:    &stdout,
			Error:     &stderr,
		},
	)

	if exitCode != 0 {
		t.Fatalf("expected exit zero, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got:\n%s", stderr.String())
	}
	if strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("expected JSON mode not to emit human help, got:\n%s", stdout.String())
	}

	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected started and completed events, got %d lines", len(lines))
	}
	for index, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatalf("line %d is not valid JSON: %s", index+1, line)
		}
	}
}

func TestJSONDoctorCommandNameSkipsConfigFlagValue(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	projectRoot := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(projectRoot, "composer.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatalf("write Composer fixture: %v", err)
	}
	configPath := filepath.Join(projectRoot, "custom.toml")
	if err := os.WriteFile(
		configPath,
		[]byte("schema_version = 1\n"),
		0o644,
	); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{Application: app.New(app.Dependencies{
			Providers: testProviders(),
		})},
		cli.Execution{
			Arguments: []string{
				"--json",
				"--project",
				projectRoot,
				"--config",
				configPath,
				"doctor",
			},
			Input:  strings.NewReader(""),
			Output: &stdout,
			Error:  &stderr,
		},
	)
	if exitCode != 0 {
		t.Fatalf("expected exit zero, got %d\n%s", exitCode, stdout.String())
	}

	var event model.Event
	firstLine, _, _ := strings.Cut(stdout.String(), "\n")
	if err := json.Unmarshal([]byte(firstLine), &event); err != nil {
		t.Fatalf("decode started event: %v", err)
	}
	if event.Command != "doctor" {
		t.Fatalf("expected doctor command name, got %q", event.Command)
	}
}

func TestJSONDoctorExplainsNativeProviderSelection(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	nativeProvider := &testProvider{
		observation: model.ProviderObservation{
			Provider:     "native",
			Available:    true,
			Platform:     "darwin",
			Architecture: "arm64",
			Capabilities: []model.Capability{
				model.CapabilityExecuteCommand,
				model.CapabilityInspectComposer,
				model.CapabilityInspectRuntime,
			},
			Runtimes: []model.RuntimeObservation{
				{
					Name:    "php",
					Version: "8.5.0",
					SAPI:    "cli",
					Source: model.SourceReference{
						Path: "/fixture/bin/php",
						Kind: "provider_executable",
					},
				},
			},
			Composer: []model.ComposerObservation{
				{
					Version:  "2.9.5",
					Source:   "system",
					Path:     "/fixture/bin/composer",
					Identity: "sha256:composer",
				},
			},
			Extensions:  []model.ExtensionObservation{},
			Diagnostics: []model.Diagnostic{},
			Fingerprint: "sha256:native",
		},
	}
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			context.Context,
			discovery.Request,
		) (model.ProjectFacts, error) {
			return cliCompatibleFacts(), nil
		},
		Providers: providerSet(nativeProvider),
	})

	exitCode := cli.Execute(
		context.Background(),
		cli.Dependencies{Application: application},
		cli.Execution{
			Arguments: []string{
				"--json",
				"--provider",
				"native",
				"doctor",
			},
			Input:  strings.NewReader(""),
			Output: &stdout,
			Error:  &stderr,
		},
	)
	if exitCode != 0 {
		t.Fatalf(
			"expected doctor exit zero, got %d\n%s",
			exitCode,
			stdout.String(),
		)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected JSON stderr to be empty, got:\n%s", stderr.String())
	}

	var events []struct {
		Command string          `json:"command"`
		Type    model.EventType `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		var event struct {
			Command string          `json:"command"`
			Type    model.EventType `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode doctor event: %v", err)
		}
		events = append(events, event)
	}
	expectedTypes := []model.EventType{
		model.EventStarted,
		model.EventFact,
		model.EventFact,
		model.EventPlan,
		model.EventCompleted,
	}
	if len(events) != len(expectedTypes) {
		t.Fatalf("unexpected doctor events %#v", events)
	}
	for index, expected := range expectedTypes {
		if events[index].Command != "doctor" ||
			events[index].Type != expected {
			t.Fatalf("unexpected doctor event %d: %#v", index+1, events[index])
		}
	}
	var builtPlan model.Plan
	if err := json.Unmarshal(events[3].Payload, &builtPlan); err != nil {
		t.Fatalf("decode doctor plan: %v", err)
	}
	if builtPlan.Provider.Name != "native" ||
		builtPlan.Provider.Reason != "explicit" ||
		builtPlan.Operation != model.OperationDoctor {
		t.Fatalf("unexpected doctor plan %#v", builtPlan)
	}
}

func readEventGolden(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "golden", "events", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}

	return string(content)
}
