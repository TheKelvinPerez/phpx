//go:build integration

package ddev_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
	"github.com/elefantephp/elefante/internal/providers/ddev"
)

const ddevIntegrationProject = "elefante-phase9-integration"

func TestDDEVIntegration(t *testing.T) {
	if os.Getenv("ELEFANTE_DDEV_INTEGRATION") != "1" {
		t.Skip("set ELEFANTE_DDEV_INTEGRATION=1 to run the isolated DDEV integration")
	}
	ddevPath, err := exec.LookPath("ddev")
	if err != nil {
		t.Fatalf("DDEV integration requires ddev: %v", err)
	}
	fixtureRoot, err := filepath.Abs(
		filepath.Join("testdata", "integration", "project"),
	)
	if err != nil {
		t.Fatalf("resolve DDEV integration fixture: %v", err)
	}
	projectRoot := filepath.Join(t.TempDir(), "project")
	copyDDEVIntegrationFixture(t, fixtureRoot, projectRoot)

	initial, err := ddevProjectStates(t.Context(), ddevPath)
	if err != nil {
		t.Fatalf("list DDEV projects before fixture cleanup: %v", err)
	}
	if _, found := initial[ddevIntegrationProject]; found {
		if err := deleteDDEVProject(
			t.Context(),
			ddevPath,
			ddevIntegrationProject,
		); err != nil {
			t.Fatalf("remove stale DDEV integration fixture: %v", err)
		}
	}
	baseline, err := ddevProjectStates(t.Context(), ddevPath)
	if err != nil {
		t.Fatalf("list DDEV projects before integration: %v", err)
	}
	defer func() {
		cleanupContext, cancel := context.WithTimeout(
			context.Background(),
			2*time.Minute,
		)
		defer cancel()

		if err := deleteDDEVProject(
			cleanupContext,
			ddevPath,
			ddevIntegrationProject,
		); err != nil {
			t.Errorf("remove DDEV integration fixture: %v", err)
		}
		after, err := ddevProjectStates(cleanupContext, ddevPath)
		if err != nil {
			t.Errorf("list DDEV projects after integration: %v", err)
			return
		}
		if !reflect.DeepEqual(after, baseline) {
			t.Errorf(
				"DDEV integration changed projects outside its fixture\nbefore: %#v\nafter:  %#v",
				baseline,
				after,
			)
		}
	}()

	if _, err := runDDEV(
		t.Context(),
		ddevPath,
		projectRoot,
		"--skip-hooks",
		"start",
		"--skip-confirmation",
	); err != nil {
		t.Fatalf("start DDEV integration fixture: %v", err)
	}

	application := app.New(app.Dependencies{
		Providers: []providers.Provider{
			ddev.New(),
		},
	})
	doctor, err := application.Doctor(t.Context(), app.DoctorRequest{
		ProjectPath: projectRoot,
		Provider:    "ddev",
	})
	if err != nil {
		t.Fatalf("run DDEV doctor integration: %v", err)
	}
	observation := observationForProvider(t, doctor.Observations, "ddev")
	assertRealDDEVObservation(t, observation)
	if doctor.Plan.Provider.Name != "ddev" ||
		doctor.Plan.Operation != model.OperationDoctor ||
		len(doctor.Plan.Actions) != 0 {
		t.Fatalf("unexpected DDEV doctor plan %#v", doctor.Plan)
	}

	synchronization, err := application.Plan(t.Context(), app.PlanRequest{
		ProjectPath: projectRoot,
		Provider:    "ddev",
	})
	if err != nil {
		t.Fatalf("run DDEV plan integration: %v", err)
	}
	if synchronization.Plan.Provider.Name != "ddev" ||
		synchronization.Plan.Operation != model.OperationSync ||
		!strings.HasPrefix(synchronization.Plan.Digest, "sha256:") {
		t.Fatalf("unexpected DDEV synchronization plan %#v", synchronization.Plan)
	}
	for _, action := range synchronization.Plan.Actions {
		if action.Kind == model.ActionPrepareProvider {
			t.Fatalf("running DDEV fixture must not plan provider start: %#v", action)
		}
	}
}

func copyDDEVIntegrationFixture(
	t *testing.T,
	sourceRoot string,
	targetRoot string,
) {
	t.Helper()

	for _, relativePath := range []string{
		filepath.Join(".ddev", "config.yaml"),
		"composer.json",
		filepath.Join("public", "index.php"),
	} {
		content, err := os.ReadFile(filepath.Join(sourceRoot, relativePath))
		if err != nil {
			t.Fatalf("read DDEV integration fixture %s: %v", relativePath, err)
		}
		targetPath := filepath.Join(targetRoot, relativePath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			t.Fatalf("create DDEV fixture directory: %v", err)
		}
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			t.Fatalf("copy DDEV integration fixture %s: %v", relativePath, err)
		}
	}
}

func assertRealDDEVObservation(
	t *testing.T,
	observation model.ProviderObservation,
) {
	t.Helper()

	if !observation.Available ||
		observation.Version == "" ||
		observation.Platform != "darwin" ||
		observation.Architecture != "arm64" ||
		observation.State != model.ProviderStateRunning {
		t.Fatalf("unexpected real DDEV identity %#v", observation)
	}
	if len(observation.Engines) != 1 ||
		observation.Engines[0].Name != "docker" ||
		observation.Engines[0].Version == "" ||
		observation.Engines[0].Platform != "orbstack" {
		t.Fatalf("expected DDEV through OrbStack, got %#v", observation.Engines)
	}
	if len(observation.Runtimes) != 1 ||
		observation.Runtimes[0].Name != "php" ||
		!strings.HasPrefix(observation.Runtimes[0].Version, "8.3.") ||
		observation.Runtimes[0].SAPI != "cli" {
		t.Fatalf("unexpected real DDEV PHP %#v", observation.Runtimes)
	}
	if !hasObservedExtension(observation.Extensions, "ext-json") {
		t.Fatalf("real DDEV PHP did not report ext-json: %#v", observation.Extensions)
	}
	if len(observation.Composer) != 1 ||
		observation.Composer[0].Version == "" ||
		observation.Composer[0].Path == "" ||
		observation.Composer[0].Identity == "" ||
		observation.Composer[0].Source != "ddev" {
		t.Fatalf("unexpected real DDEV Composer %#v", observation.Composer)
	}
}

func observationForProvider(
	t *testing.T,
	observations []model.ProviderObservation,
	name string,
) model.ProviderObservation {
	t.Helper()

	for _, observation := range observations {
		if observation.Provider == name {
			return observation
		}
	}
	t.Fatalf("expected provider observation %q, got %#v", name, observations)

	return model.ProviderObservation{}
}

func hasObservedExtension(
	extensions []model.ExtensionObservation,
	name string,
) bool {
	for _, extension := range extensions {
		if extension.Name == name && extension.Available {
			return true
		}
	}

	return false
}

func ddevProjectStates(
	ctx context.Context,
	ddevPath string,
) (map[string]string, error) {
	output, err := runDDEV(
		ctx,
		ddevPath,
		"",
		"list",
		"--json-output",
	)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Raw []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"raw"`
	}
	if err := json.Unmarshal(output, &envelope); err != nil {
		return nil, fmt.Errorf("decode DDEV project list: %w", err)
	}
	states := make(map[string]string, len(envelope.Raw))
	for _, project := range envelope.Raw {
		states[project.Name] = project.Status
	}

	return states, nil
}

func deleteDDEVProject(
	ctx context.Context,
	ddevPath string,
	name string,
) error {
	states, err := ddevProjectStates(ctx, ddevPath)
	if err != nil {
		return err
	}
	if _, found := states[name]; !found {
		return nil
	}
	_, err = runDDEV(
		ctx,
		ddevPath,
		"",
		"--skip-hooks",
		"delete",
		"--omit-snapshot",
		"--yes",
		"--clean-containers=false",
		name,
	)

	return err
}

func runDDEV(
	ctx context.Context,
	ddevPath string,
	directory string,
	arguments ...string,
) ([]byte, error) {
	command := exec.CommandContext(ctx, ddevPath, arguments...)
	command.Dir = directory
	command.Env = append(
		os.Environ(),
		"DDEV_NO_INSTRUMENTATION=true",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf(
			"ddev %s: %w\nstdout:\n%s\nstderr:\n%s",
			strings.Join(arguments, " "),
			err,
			stdout.String(),
			stderr.String(),
		)
	}

	return stdout.Bytes(), nil
}
