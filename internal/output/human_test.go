package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/output"
	"github.com/elefantephp/elefante/internal/security"
)

func TestHumanRendererKeepsResultsAndErrorsOnSeparateStreams(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := output.NewHumanRenderer(&stdout, &stderr)

	if err := renderer.Started(); err != nil {
		t.Fatalf("render started lifecycle: %v", err)
	}
	if err := renderer.Result(output.Result{Text: "elefante dev"}); err != nil {
		t.Fatalf("render result: %v", err)
	}

	commandError := model.NewError(
		model.ErrorUsage,
		`unknown command "unknown" for "elefante"`,
	).WithHint("Run elefante --help to see available commands.")
	if err := renderer.Error(commandError); err != nil {
		t.Fatalf("render error: %v", err)
	}
	if err := renderer.Completed(model.ExitForError(commandError)); err != nil {
		t.Fatalf("render completed lifecycle: %v", err)
	}

	if got, expected := stdout.String(), "elefante dev\n"; got != expected {
		t.Fatalf("expected human result %q, got %q", expected, got)
	}
	expectedError := "Error: unknown command \"unknown\" for \"elefante\"\nHint: Run elefante --help to see available commands.\n"
	if stderr.String() != expectedError {
		t.Fatalf("expected human error %q, got %q", expectedError, stderr.String())
	}
}

func TestHumanRendererKeepsPlanAndDiagnosticsOnSeparateStreams(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := output.NewHumanRenderer(&stdout, &stderr)

	if err := renderer.Diagnostic(output.Diagnostic{
		Payload: model.Diagnostic{
			Code:     "ELEFANTE_REQUIREMENT_INCOMPATIBLE",
			Severity: model.SeverityError,
			Message:  "The native PHP runtime is incompatible.",
		},
		Text: "Error [ELEFANTE_REQUIREMENT_INCOMPATIBLE]: The native PHP runtime is incompatible.",
	}); err != nil {
		t.Fatalf("render diagnostic: %v", err)
	}
	if err := renderer.Plan(output.Plan{
		Payload: model.Plan{
			SchemaVersion: model.PlanSchemaVersion,
			Operation:     model.OperationSync,
		},
		Text: "Provider: native\nReason: only available",
	}); err != nil {
		t.Fatalf("render plan: %v", err)
	}

	if got, expected := stdout.String(), "Provider: native\nReason: only available\n"; got != expected {
		t.Fatalf("expected human plan %q, got %q", expected, got)
	}
	expectedError := "Error [ELEFANTE_REQUIREMENT_INCOMPATIBLE]: The native PHP runtime is incompatible.\n"
	if stderr.String() != expectedError {
		t.Fatalf("expected human diagnostic %q, got %q", expectedError, stderr.String())
	}
}

func TestHumanRendererRedactsEveryOutputStream(t *testing.T) {
	t.Parallel()

	const secret = "human-output-secret"
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := output.NewHumanRendererWithRedactor(
		&stdout,
		&stderr,
		security.NewRedactor(secret),
	)

	if err := renderer.Result(output.Result{
		Text: "Result token: " + secret,
	}); err != nil {
		t.Fatalf("render result: %v", err)
	}
	if err := renderer.Error(model.NewError(
		model.ErrorNetwork,
		"Authorization: Bearer "+secret,
	).WithHint("Remove " + secret + " from the configuration.")); err != nil {
		t.Fatalf("render error: %v", err)
	}

	captured := stdout.String() + stderr.String()
	if strings.Contains(captured, secret) {
		t.Fatalf("human output leaked a secret: %s", captured)
	}
	if strings.Count(captured, security.Redacted) < 3 {
		t.Fatalf("expected all secret occurrences to be redacted: %s", captured)
	}
}
