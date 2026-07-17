package output_test

import (
	"bytes"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/output"
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
