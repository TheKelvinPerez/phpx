package native_test

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/elefantephp/elefante/internal/executor"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
	"github.com/elefantephp/elefante/internal/providers/native"
	"github.com/elefantephp/elefante/internal/providers/providertest"
)

func TestInspectReportsMissingPHP(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"composer": "/fixture/bin/composer",
		},
		outputs: map[string][]byte{
			"/fixture/bin/composer": []byte(
				"Composer version 2.9.5 2026-01-29 11:40:53\n",
			),
		},
	}
	provider := native.New(native.Dependencies{
		Runner:       runner,
		GOOS:         "darwin",
		GOARCH:       "arm64",
		ProviderPath: "/fixture/bin/elefante",
	})

	observation, err := provider.Inspect(
		t.Context(),
		providers.InspectRequest{},
	)
	if err != nil {
		t.Fatalf("inspect native provider: %v", err)
	}

	if !observation.Available {
		t.Fatalf("expected native topology to remain observable: %#v", observation)
	}
	if len(observation.Runtimes) != 0 {
		t.Fatalf("expected no PHP runtime, got %#v", observation.Runtimes)
	}
	diagnostic := diagnosticByCode(
		t,
		observation.Diagnostics,
		"ELEFANTE_NATIVE_PHP_NOT_FOUND",
	)
	if diagnostic.Severity != model.SeverityError {
		t.Fatalf("expected a blocking PHP diagnostic, got %#v", diagnostic)
	}
	if len(observation.Composer) != 1 ||
		observation.Composer[0].Version != "2.9.5" {
		t.Fatalf("expected Composer inspection to continue, got %#v", observation)
	}
	if !runner.lookedUp("php") || !runner.lookedUp("composer") {
		t.Fatalf("expected PHP and Composer discovery, got %#v", runner.lookups)
	}
}

func TestInspectReportsCompatiblePHPAndExtensions(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"php":      "/fixture/bin/php",
			"composer": "/fixture/bin/composer",
		},
		outputs: map[string][]byte{
			"/fixture/bin/php": []byte(
				`{"version":"8.5.0","sapi":"cli","binary":"/fixture/bin/php","extensions":[{"name":"json","version":"8.5.0"},{"name":"curl","version":"8.5.0"}]}`,
			),
			"/fixture/bin/composer": []byte(
				"Composer version 2.9.5 2026-01-29 11:40:53\n",
			),
		},
	}
	provider := native.New(native.Dependencies{
		Runner:       runner,
		GOOS:         "darwin",
		GOARCH:       "arm64",
		ProviderPath: "/fixture/bin/elefante",
	})

	observation, err := provider.Inspect(
		t.Context(),
		providers.InspectRequest{
			Project: model.ProjectIdentity{
				ComposerRoot: "/workspace",
			},
		},
	)
	if err != nil {
		t.Fatalf("inspect native provider: %v", err)
	}

	if len(observation.Diagnostics) != 0 {
		t.Fatalf("expected compatible inspection, got %#v", observation.Diagnostics)
	}
	if observation.Platform != "darwin" ||
		observation.Architecture != "arm64" {
		t.Fatalf("expected native platform identity, got %#v", observation)
	}
	if len(observation.Runtimes) != 1 ||
		observation.Runtimes[0].Name != "php" ||
		observation.Runtimes[0].Version != "8.5.0" ||
		observation.Runtimes[0].SAPI != "cli" ||
		observation.Runtimes[0].Source.Path != "/fixture/bin/php" {
		t.Fatalf("unexpected PHP observation %#v", observation.Runtimes)
	}
	if len(observation.Extensions) != 2 ||
		observation.Extensions[0].Name != "ext-curl" ||
		observation.Extensions[1].Name != "ext-json" {
		t.Fatalf("expected sorted extension observations, got %#v", observation.Extensions)
	}
	for _, extension := range observation.Extensions {
		if !extension.Available ||
			extension.Source.Path != "/fixture/bin/php" {
			t.Fatalf("unexpected extension provenance %#v", extension)
		}
	}
	if hasCapability(
		observation.Capabilities,
		model.CapabilityInstallRuntime,
	) || hasCapability(
		observation.Capabilities,
		model.CapabilityInstallExtension,
	) {
		t.Fatalf("native inspection must not advertise installation: %#v", observation.Capabilities)
	}
	if len(observation.Fingerprint) != 71 ||
		observation.Fingerprint[:7] != "sha256:" {
		t.Fatalf("expected SHA256 observation fingerprint, got %q", observation.Fingerprint)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("expected PHP and Composer probes, got %#v", runner.commands)
	}
	phpCommand := runner.commands[0]
	if phpCommand.Executable != "/fixture/bin/php" ||
		len(phpCommand.Arguments) != 2 ||
		phpCommand.Arguments[0] != "-r" ||
		phpCommand.WorkingDirectory != "" {
		t.Fatalf("unexpected safe PHP probe %#v", phpCommand)
	}
}

func TestInspectNormalizesExtensionsAsComposerPlatformPackages(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"php":      "/fixture/bin/php",
			"composer": "/fixture/bin/composer",
		},
		outputs: map[string][]byte{
			"/fixture/bin/php": []byte(
				`{"version":"8.5.0","sapi":"cli","binary":"/fixture/bin/php","extensions":[{"name":"Zend OPcache","version":"8.5.0"},{"name":"PDO_SQLITE","version":"8.5.0"}]}`,
			),
			"/fixture/bin/composer": []byte(
				"Composer version 2.9.5 2026-01-29 11:40:53\n",
			),
		},
	}
	provider := native.New(native.Dependencies{
		Runner: runner,
		GOOS:   "darwin",
		GOARCH: "arm64",
	})

	observation, err := provider.Inspect(
		t.Context(),
		providers.InspectRequest{},
	)
	if err != nil {
		t.Fatalf("inspect native provider: %v", err)
	}

	if len(observation.Extensions) != 2 ||
		observation.Extensions[0].Name != "ext-pdo_sqlite" ||
		observation.Extensions[1].Name != "ext-zend-opcache" {
		t.Fatalf(
			"expected Composer platform extension names, got %#v",
			observation.Extensions,
		)
	}
}

func TestInspectRejectsTrailingPHPInspectionData(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"php":      "/fixture/bin/php",
			"composer": "/fixture/bin/composer",
		},
		outputs: map[string][]byte{
			"/fixture/bin/php": []byte(
				`{"version":"8.5.0","sapi":"cli","binary":"/fixture/bin/php","extensions":[]} {}`,
			),
			"/fixture/bin/composer": []byte(
				"Composer version 2.9.5 2026-01-29 11:40:53\n",
			),
		},
	}
	provider := native.New(native.Dependencies{
		Runner: runner,
		GOOS:   "darwin",
		GOARCH: "arm64",
	})

	_, err := provider.Inspect(t.Context(), providers.InspectRequest{})
	if err == nil {
		t.Fatal("expected malformed PHP inspection output to fail")
	}
	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed provider error, got %T: %v", err, err)
	}
	if commandError.Code != model.ErrorProvider ||
		commandError.Provider != "native" {
		t.Fatalf("unexpected provider error %#v", commandError)
	}
}

func TestInspectPreservesPHPProcessFailure(t *testing.T) {
	processFailure := errors.New("php process failed")
	runner := &fakeRunner{
		paths: map[string]string{
			"php":      "/fixture/bin/php",
			"composer": "/fixture/bin/composer",
		},
		errors: map[string]error{
			"/fixture/bin/php": processFailure,
		},
	}
	provider := native.New(native.Dependencies{
		Runner: runner,
		GOOS:   "darwin",
		GOARCH: "arm64",
	})

	_, err := provider.Inspect(t.Context(), providers.InspectRequest{})
	if err == nil {
		t.Fatal("expected PHP process failure")
	}
	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed provider error, got %T: %v", err, err)
	}
	if commandError.Code != model.ErrorProvider ||
		commandError.Provider != "native" ||
		!commandError.Retryable {
		t.Fatalf("unexpected provider error %#v", commandError)
	}
	if !errors.Is(err, processFailure) {
		t.Fatalf("expected process cause to be preserved, got %v", err)
	}
}

func TestInspectReportsMissingComposer(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"php": "/fixture/bin/php",
		},
		outputs: map[string][]byte{
			"/fixture/bin/php": []byte(
				`{"version":"8.5.0","sapi":"cli","binary":"/fixture/bin/php","extensions":[]}`,
			),
		},
	}
	provider := native.New(native.Dependencies{
		Runner: runner,
		GOOS:   "darwin",
		GOARCH: "arm64",
	})

	observation, err := provider.Inspect(
		t.Context(),
		providers.InspectRequest{},
	)
	if err != nil {
		t.Fatalf("inspect native provider: %v", err)
	}

	if len(observation.Runtimes) != 1 {
		t.Fatalf("expected PHP inspection to continue, got %#v", observation)
	}
	if len(observation.Composer) != 0 {
		t.Fatalf("expected no Composer observation, got %#v", observation.Composer)
	}
	diagnostic := diagnosticByCode(
		t,
		observation.Diagnostics,
		"ELEFANTE_NATIVE_COMPOSER_NOT_FOUND",
	)
	if diagnostic.Severity != model.SeverityError ||
		diagnostic.Provider != "native" {
		t.Fatalf("unexpected Composer diagnostic %#v", diagnostic)
	}
}

func TestInspectPreservesComposerProcessFailure(t *testing.T) {
	processFailure := errors.New("composer process failed")
	runner := &fakeRunner{
		paths: map[string]string{
			"php":      "/fixture/bin/php",
			"composer": "/fixture/bin/composer",
		},
		outputs: map[string][]byte{
			"/fixture/bin/php": []byte(
				`{"version":"8.5.0","sapi":"cli","binary":"/fixture/bin/php","extensions":[]}`,
			),
		},
		errors: map[string]error{
			"/fixture/bin/composer": processFailure,
		},
	}
	provider := native.New(native.Dependencies{
		Runner: runner,
		GOOS:   "darwin",
		GOARCH: "arm64",
	})

	_, err := provider.Inspect(t.Context(), providers.InspectRequest{})
	if err == nil {
		t.Fatal("expected Composer process failure")
	}
	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed provider error, got %T: %v", err, err)
	}
	if commandError.Code != model.ErrorProvider ||
		commandError.Provider != "native" ||
		!commandError.Retryable {
		t.Fatalf("unexpected provider error %#v", commandError)
	}
	if !errors.Is(err, processFailure) {
		t.Fatalf("expected process cause to be preserved, got %v", err)
	}
}

func TestProviderConformance(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]string{
			"php":      "/fixture/bin/php",
			"composer": "/fixture/bin/composer",
		},
		outputs: map[string][]byte{
			"/fixture/bin/php": []byte(
				`{"version":"8.5.0","sapi":"cli","binary":"/fixture/bin/php","extensions":[{"name":"json","version":"8.5.0"}]}`,
			),
			"/fixture/bin/composer": []byte(
				"Composer version 2.9.5 2026-01-29 11:40:53\n",
			),
		},
	}

	providertest.Run(t, providertest.Suite{
		Provider: native.New(native.Dependencies{
			Runner:       runner,
			GOOS:         "darwin",
			GOARCH:       "arm64",
			ProviderPath: "/fixture/bin/elefante",
		}),
		InspectRequest: providers.InspectRequest{
			Project: model.ProjectIdentity{
				ComposerRoot: "/workspace",
			},
			Offline: true,
		},
		ExecutionRequest: providers.ExecutionRequest{
			Executable:       "php",
			Arguments:        []string{"-r", `echo "safe";`},
			WorkingDirectory: "/workspace",
			Environment:      []string{"APP_ENV=test"},
		},
	})
}

type fakeRunner struct {
	paths    map[string]string
	outputs  map[string][]byte
	errors   map[string]error
	lookups  []string
	commands []executor.Command
}

func hasCapability(
	capabilities []model.Capability,
	expected model.Capability,
) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}

	return false
}

func (runner *fakeRunner) LookPath(file string) (string, error) {
	runner.lookups = append(runner.lookups, file)
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
	if err := runner.errors[command.Executable]; err != nil {
		return nil, err
	}
	if output, found := runner.outputs[command.Executable]; found {
		return append([]byte(nil), output...), nil
	}

	return nil, errors.New("unexpected command")
}

func (runner *fakeRunner) lookedUp(executable string) bool {
	for _, lookup := range runner.lookups {
		if lookup == executable {
			return true
		}
	}

	return false
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
