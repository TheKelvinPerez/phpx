package output_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/output"
	"github.com/elefantephp/elefante/internal/security"
)

func TestMachineRendererSuccessSequence(t *testing.T) {
	first := renderVersionSuccess(t)
	second := renderVersionSuccess(t)

	if first != second {
		t.Fatalf("expected equivalent inputs to produce identical events\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	expected := readGolden(t, "version-success.ndjson")
	if first != expected {
		t.Fatalf("machine event sequence does not match golden\nexpected:\n%s\ngot:\n%s", expected, first)
	}

	assertValidJSONLines(t, first, 3)
}

func TestMachineRendererErrorSequence(t *testing.T) {
	var buffer bytes.Buffer
	renderer := output.NewMachineRenderer(&buffer, "unknown")
	commandError := model.NewError(
		model.ErrorUsage,
		`unknown command "unknown" for "elefante"`,
	).WithHint("Run elefante --help to see available commands.")

	if err := renderer.Started(); err != nil {
		t.Fatalf("render started event: %v", err)
	}
	if err := renderer.Error(commandError); err != nil {
		t.Fatalf("render error event: %v", err)
	}
	if err := renderer.Completed(model.ExitForError(commandError)); err != nil {
		t.Fatalf("render completed event: %v", err)
	}

	expected := readGolden(t, "usage-error.ndjson")
	if buffer.String() != expected {
		t.Fatalf(
			"machine error sequence does not match golden\nexpected:\n%s\ngot:\n%s",
			expected,
			buffer.String(),
		)
	}

	assertValidJSONLines(t, buffer.String(), 3)
}

func TestMachineRendererDoesNotWriteMalformedPartialJSON(t *testing.T) {
	var buffer bytes.Buffer
	renderer := output.NewMachineRenderer(&buffer, "version")

	err := renderer.Result(output.Result{
		Payload: make(chan int),
	})
	if err == nil {
		t.Fatal("expected an unsupported payload to fail JSON encoding")
	}
	if buffer.Len() != 0 {
		t.Fatalf("expected no partial JSON output, got %q", buffer.String())
	}

	if err := renderer.Started(); err != nil {
		t.Fatalf("render event after encoding failure: %v", err)
	}
	expected := "{\"schema\":\"elefante.events/v1\",\"sequence\":1,\"command\":\"version\",\"type\":\"started\",\"payload\":{}}\n"
	if buffer.String() != expected {
		t.Fatalf("expected failed encoding not to consume a sequence number\nexpected: %q\ngot:      %q", expected, buffer.String())
	}
}

func TestMachineRendererRedactsEveryEventPayload(t *testing.T) {
	t.Parallel()

	const secret = "machine-output-secret"
	var buffer bytes.Buffer
	renderer := output.NewMachineRendererWithRedactor(
		&buffer,
		"plan",
		security.NewRedactor(secret),
	)

	if err := renderer.Result(output.Result{
		Payload: map[string]any{
			"message":       "resolved " + secret,
			"authorization": "Bearer " + secret,
		},
	}); err != nil {
		t.Fatalf("render result: %v", err)
	}

	if strings.Contains(buffer.String(), secret) {
		t.Fatalf("machine output leaked a secret: %s", buffer.String())
	}
	if !strings.Contains(buffer.String(), security.Redacted) {
		t.Fatalf("expected a redaction marker, got %s", buffer.String())
	}
	assertValidJSONLines(t, buffer.String(), 1)
}

func TestMachineRendererEmitsDiagnosticAndPlanEvents(t *testing.T) {
	var buffer bytes.Buffer
	renderer := output.NewMachineRenderer(&buffer, "plan")
	diagnostic := model.Diagnostic{
		Code:     "ELEFANTE_REQUIREMENT_INCOMPATIBLE",
		Severity: model.SeverityError,
		Message:  "The native PHP runtime is incompatible.",
		Provider: "native",
	}
	builtPlan := model.Plan{
		SchemaVersion: model.PlanSchemaVersion,
		Operation:     model.OperationSync,
		Requirements:  []model.RequirementResolution{},
		Actions:       []model.PlanAction{},
		Inputs:        []model.InputFingerprint{},
		Policy:        model.PlanPolicy{},
		Digest:        "sha256:plan",
	}

	if err := renderer.Started(); err != nil {
		t.Fatalf("render started event: %v", err)
	}
	if err := renderer.Diagnostic(output.Diagnostic{
		Payload: diagnostic,
		Text:    diagnostic.Message,
	}); err != nil {
		t.Fatalf("render diagnostic event: %v", err)
	}
	if err := renderer.Plan(output.Plan{
		Payload: builtPlan,
		Text:    "Provider: native",
	}); err != nil {
		t.Fatalf("render plan event: %v", err)
	}
	if err := renderer.Completed(model.Exit{
		Origin: model.ExitOriginElefante,
		Code:   0,
	}); err != nil {
		t.Fatalf("render completed event: %v", err)
	}

	var events []struct {
		Type    model.EventType `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	for _, line := range strings.Split(strings.TrimSpace(buffer.String()), "\n") {
		var event struct {
			Type    model.EventType `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		events = append(events, event)
	}
	if len(events) != 4 ||
		events[1].Type != model.EventDiagnostic ||
		events[2].Type != model.EventPlan {
		t.Fatalf("unexpected event sequence %#v", events)
	}
}

func renderVersionSuccess(t *testing.T) string {
	t.Helper()

	var buffer bytes.Buffer
	renderer := output.NewMachineRenderer(&buffer, "version")

	if err := renderer.Started(); err != nil {
		t.Fatalf("render started event: %v", err)
	}
	if err := renderer.Result(output.Result{
		Payload: model.BuildInfo{Version: "dev"},
	}); err != nil {
		t.Fatalf("render result event: %v", err)
	}
	if err := renderer.Completed(model.Exit{
		Origin: model.ExitOriginElefante,
		Code:   0,
	}); err != nil {
		t.Fatalf("render completed event: %v", err)
	}

	return buffer.String()
}

func readGolden(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "golden", "events", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}

	return string(content)
}

func assertValidJSONLines(t *testing.T, content string, expectedLines int) {
	t.Helper()

	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	if len(lines) != expectedLines {
		t.Fatalf("expected %d event lines, got %d", expectedLines, len(lines))
	}

	for index, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatalf("event line %d is not valid JSON: %s", index+1, line)
		}
	}
}
