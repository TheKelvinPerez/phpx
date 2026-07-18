package cli_test

import (
	"context"
	"errors"
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
		Providers: testProviders(),
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

func TestPlanRendersNativeDecisionAndForwardsReadOnlyPolicy(t *testing.T) {
	var received discovery.Request
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
			_ context.Context,
			request discovery.Request,
		) (model.ProjectFacts, error) {
			received = request

			return cliCompatibleFacts(), nil
		},
		Providers: providerSet(nativeProvider),
	})

	_, stdout, stderr, err := executeCommand(
		t,
		context.Background(),
		strings.NewReader(""),
		cli.Dependencies{Application: application},
		"--project",
		"/workspace",
		"--config",
		"/workspace/elefante.toml",
		"--provider",
		"native",
		"--offline",
		"--frozen",
		"plan",
	)
	var commandError *model.Error
	if !errors.As(err, &commandError) ||
		commandError.Code != model.ErrorRequirements {
		t.Fatalf(
			"expected offline requirements blocker, got %v\nstdout:\n%s\nstderr:\n%s",
			err,
			stdout,
			stderr,
		)
	}
	if received.StartPath != "/workspace" ||
		received.ConfigPath != "/workspace/elefante.toml" {
		t.Fatalf("unexpected discovery request %#v", received)
	}
	if len(nativeProvider.inspectRequests) != 1 ||
		!nativeProvider.inspectRequests[0].Offline {
		t.Fatalf(
			"expected offline provider inspection, got %#v",
			nativeProvider.inspectRequests,
		)
	}
	for _, expected := range []string{
		"Project: /workspace",
		"Provider: native",
		"Selection reason: explicit",
		"PHP: 8.5.0, SAPI cli, /fixture/bin/php",
		"Composer: 2.9.5, /fixture/bin/composer",
		"Operation: sync",
		"Plan digest: sha256:",
		"Next command: elefante sync",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected plan output to contain %q, got:\n%s", expected, stdout)
		}
	}
	if !strings.Contains(
		stderr,
		"ELEFANTE_OFFLINE_NETWORK_REQUIRED",
	) {
		t.Fatalf("expected offline diagnostic, got:\n%s", stderr)
	}
}

func cliCompatibleFacts() model.ProjectFacts {
	return model.ProjectFacts{
		Identity: model.ProjectIdentity{
			ComposerRoot:    "/workspace",
			ApplicationRoot: "/workspace",
			WorkspaceRoot:   "/workspace",
			IdentityKey:     "sha256:project",
		},
		Composer: model.ComposerFacts{
			Manifest: model.ComposerManifestFacts{
				Path: "/workspace/composer.json",
				Name: "acme/example",
			},
			Lock: model.ComposerLockFacts{
				Path:   "/workspace/composer.lock",
				Status: model.ComposerLockFresh,
			},
			PlatformRequirements: []model.Requirement{
				{
					Name:       "php",
					Kind:       model.RequirementPHP,
					Constraint: "^8.4",
					Scope:      model.RequirementScopeRoot,
					Sources: []model.SourceReference{
						{
							Path:  "/workspace/composer.json",
							Kind:  "composer_manifest",
							Field: "/require/php",
						},
					},
				},
			},
		},
		InputFingerprints: []model.InputFingerprint{
			{
				Path:   "/workspace/composer.json",
				Kind:   "composer_manifest",
				SHA256: "manifest",
				Size:   128,
			},
		},
	}
}
