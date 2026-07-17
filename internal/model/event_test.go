package model_test

import (
	"testing"

	"github.com/elefantephp/elefante/internal/model"
)

func TestMachineEventSchemaAndTypeNamesRemainStable(t *testing.T) {
	if model.EventSchema != "elefante.events/v1" {
		t.Fatalf("expected event schema elefante.events/v1, got %q", model.EventSchema)
	}

	eventTypes := []model.EventType{
		model.EventStarted,
		model.EventFact,
		model.EventDiagnostic,
		model.EventPlan,
		model.EventApprovalRequired,
		model.EventProgress,
		model.EventStdout,
		model.EventStderr,
		model.EventResult,
		model.EventError,
		model.EventCompleted,
	}
	expected := []string{
		"started",
		"fact",
		"diagnostic",
		"plan",
		"approval_required",
		"progress",
		"stdout",
		"stderr",
		"result",
		"error",
		"completed",
	}

	for index, eventType := range eventTypes {
		if string(eventType) != expected[index] {
			t.Fatalf("expected event type %q, got %q", expected[index], eventType)
		}
	}
}
