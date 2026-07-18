package cli_test

import (
	"context"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/cli"
	"github.com/elefantephp/elefante/internal/discovery"
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

func TestDoctorForwardsExplicitProjectAndConfigPaths(t *testing.T) {
	var received discovery.Request
	application := app.New(app.Dependencies{
		DiscoverProject: func(
			_ context.Context,
			request discovery.Request,
		) (model.ProjectFacts, error) {
			received = request

			return model.ProjectFacts{
				Identity: model.ProjectIdentity{
					ComposerRoot:  "/project",
					WorkspaceRoot: "/project",
					IdentityKey:   "sha256:test",
				},
			}, nil
		},
	})

	_, _, _, err := executeCommand(
		t,
		context.Background(),
		strings.NewReader(""),
		cli.Dependencies{Application: application},
		"--project",
		"/project",
		"--config",
		"/project/custom.toml",
		"doctor",
	)
	if err != nil {
		t.Fatalf("execute doctor: %v", err)
	}
	if received.StartPath != "/project" {
		t.Fatalf("expected project path, got %#v", received)
	}
	if received.ConfigPath != "/project/custom.toml" {
		t.Fatalf("expected config path, got %#v", received)
	}
}
