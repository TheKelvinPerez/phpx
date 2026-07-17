package cli_test

import (
	"context"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/cli"
	"github.com/elefantephp/elefante/internal/model"
)

type contextKey struct{}

func TestCompleteCommandTreeExecutesInProcess(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextKey{}, "command context")
	input := strings.NewReader("command input")
	application := app.New(app.Dependencies{
		Build: model.BuildInfo{
			Version: "test-version",
		},
	})

	command, stdout, stderr, err := executeCommand(
		t,
		ctx,
		input,
		cli.Dependencies{Application: application},
		"version",
	)

	if err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if stdout != "elefante test-version\n" {
		t.Fatalf("expected injected stdout to contain version output, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected injected stderr to be empty, got %q", stderr)
	}
	if command.Context() != ctx {
		t.Fatal("expected the injected context on the complete command tree")
	}
	if command.InOrStdin() != input {
		t.Fatal("expected the injected input on the complete command tree")
	}
}
