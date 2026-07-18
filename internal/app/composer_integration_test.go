package app_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/executor"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
	"github.com/elefantephp/elefante/internal/providers/native"
)

func TestOfficialComposerInstallAndPlatformVerificationIntegration(
	t *testing.T,
) {
	if os.Getenv("ELEFANTE_COMPOSER_INTEGRATION") != "1" {
		t.Skip("set ELEFANTE_COMPOSER_INTEGRATION=1 to run official Composer integration")
	}
	phpPath, err := exec.LookPath("php")
	if err != nil {
		t.Skip("local PHP executable is unavailable")
	}
	versionOutput, err := exec.Command(
		phpPath,
		"-r",
		"echo PHP_VERSION;",
	).Output()
	if err != nil {
		t.Fatalf("inspect local PHP version: %v", err)
	}

	managerOptions := composer.ManagerOptions{
		CacheRoot: t.TempDir(),
	}
	if os.Getenv("ELEFANTE_COMPOSER_DIRECT_NETWORK") != "1" {
		managerOptions.BaseURL = officialComposerMirror(t).URL
	}
	manager := composer.NewManager(managerOptions)
	release, err := manager.Resolve(
		context.Background(),
		composer.ResolveRequest{
			Constraint: "^2",
			PHPVersion: strings.TrimSpace(string(versionOutput)),
		},
	)
	if err != nil {
		var commandError *model.Error
		_ = errors.As(err, &commandError)
		t.Fatalf(
			"resolve official Composer release: %v, transport cause: %v, details: %#v",
			err,
			errors.Unwrap(err),
			commandError,
		)
	}
	observation, err := manager.Observation(release)
	if err != nil {
		t.Fatalf("build managed Composer observation: %v", err)
	}

	projectRoot := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(projectRoot, "composer.json"),
		[]byte(`{
  "name": "elefante/composer-integration-fixture",
  "require": {
    "php": "*"
  }
}
`),
		0o644,
	); err != nil {
		t.Fatalf("write Composer integration fixture: %v", err)
	}
	prepare := model.PlanAction{
		ID:   "01-prepare-composer",
		Kind: model.ActionPrepareComposer,
		Inputs: []model.ActionInput{
			{Name: "identity", Value: release.SHA256},
			{Name: "metadata_url", Value: release.MetadataURL},
			{Name: "sha256", Value: release.SHA256},
			{Name: "url", Value: release.URL},
			{Name: "version", Value: release.Version},
		},
	}
	install := model.PlanAction{
		ID:   "02-install",
		Kind: model.ActionInstallDependencies,
		Inputs: []model.ActionInput{
			{Name: "composer", Value: release.SHA256},
			{Name: "provider", Value: "native"},
			{Name: "working_directory", Value: projectRoot},
		},
	}
	verify := model.PlanAction{
		ID:   "03-verify",
		Kind: model.ActionVerifyPlatform,
		Inputs: []model.ActionInput{
			{Name: "provider", Value: "native"},
			{Name: "working_directory", Value: projectRoot},
		},
	}
	analysis := app.Analysis{
		Observations: []model.ProviderObservation{
			{
				Provider: "native",
				Composer: []model.ComposerObservation{observation},
			},
		},
		Plan: model.Plan{
			Provider: model.ProviderSelection{Name: "native"},
			Actions:  []model.PlanAction{prepare, install, verify},
		},
	}
	nativeProvider := native.New()
	service := app.NewSyncActionService(app.SyncActionServiceDependencies{
		Providers:       []providers.Provider{nativeProvider},
		Runner:          executor.OSRunner{},
		AcquireComposer: manager.Acquire,
	})

	for _, action := range analysis.Plan.Actions {
		if _, err := service.Execute(
			t.Context(),
			app.SyncActionExecution{
				Analysis:       analysis,
				Action:         action,
				NonInteractive: true,
			},
		); err != nil {
			t.Fatalf("execute %s with official Composer: %v", action.Kind, err)
		}
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "composer.lock")); err != nil {
		t.Fatalf("official Composer did not create the fixture lock: %v", err)
	}
}

func officialComposerMirror(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if request.Method != http.MethodGet ||
				!strings.HasPrefix(request.URL.Path, "/") ||
				strings.Contains(request.URL.Path, "..") {
				http.Error(writer, "unsupported fixture request", http.StatusBadRequest)

				return
			}
			command := exec.CommandContext(
				request.Context(),
				"curl",
				"--fail",
				"--silent",
				"--show-error",
				"--location",
				"--max-time",
				"60",
				"https://getcomposer.org"+request.URL.Path,
			)
			content, err := command.CombinedOutput()
			if err != nil {
				t.Logf(
					"official Composer fixture curl failed: %v, output: %s",
					err,
					content,
				)
				http.Error(
					writer,
					fmt.Sprintf("official Composer fixture fetch failed: %v", err),
					http.StatusBadGateway,
				)

				return
			}
			switch {
			case strings.HasSuffix(request.URL.Path, ".phar"):
				writer.Header().Set("Content-Type", "application/octet-stream")
			case strings.HasSuffix(request.URL.Path, ".sha256"):
				writer.Header().Set("Content-Type", "text/plain")
			default:
				writer.Header().Set("Content-Type", "application/json")
			}
			_, _ = writer.Write(content)
		},
	))
	t.Cleanup(server.Close)

	return server
}
