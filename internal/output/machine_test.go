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
