package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/app"
	"github.com/elefantephp/elefante/internal/cli"
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
		cli.Dependencies{Application: app.New(app.Dependencies{})},
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

func readEventGolden(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "golden", "events", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}

	return string(content)
}
