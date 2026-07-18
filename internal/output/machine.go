package output

import (
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"unicode/utf8"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/security"
)

type MachineRenderer struct {
	mu       sync.Mutex
	writer   io.Writer
	command  string
	redactor security.Redactor
	sequence uint64
}

func NewMachineRenderer(writer io.Writer, command string) *MachineRenderer {
	return NewMachineRendererWithRedactor(
		writer,
		command,
		security.NewRedactor(),
	)
}

func NewMachineRendererWithRedactor(
	writer io.Writer,
	command string,
	redactor security.Redactor,
) *MachineRenderer {
	return &MachineRenderer{
		writer:   writer,
		command:  command,
		redactor: redactor,
	}
}

func (renderer *MachineRenderer) Started() error {
	return renderer.emit(model.EventStarted, struct{}{})
}

func (renderer *MachineRenderer) Fact(fact Fact) error {
	return renderer.emit(model.EventFact, fact.Payload)
}

func (renderer *MachineRenderer) Diagnostic(diagnostic Diagnostic) error {
	return renderer.emit(model.EventDiagnostic, diagnostic.Payload)
}

func (renderer *MachineRenderer) Plan(plan Plan) error {
	return renderer.emit(model.EventPlan, plan.Payload)
}

func (renderer *MachineRenderer) ApprovalRequired(
	approval ApprovalRequired,
) error {
	return renderer.emit(model.EventApprovalRequired, approval.Payload)
}

func (renderer *MachineRenderer) Result(result Result) error {
	return renderer.emit(model.EventResult, result.Payload)
}

func (renderer *MachineRenderer) Stdout(content []byte) error {
	return renderer.stream(model.EventStdout, content)
}

func (renderer *MachineRenderer) Stderr(content []byte) error {
	return renderer.stream(model.EventStderr, content)
}

func (renderer *MachineRenderer) Error(commandError *model.Error) error {
	return renderer.emit(model.EventError, commandError)
}

func (renderer *MachineRenderer) Completed(exit model.Exit) error {
	return renderer.emit(model.EventCompleted, model.CompletedPayload{
		Exit: exit,
	})
}

func (renderer *MachineRenderer) stream(
	eventType model.EventType,
	content []byte,
) error {
	payload := model.StreamPayload{
		Encoding: "utf8",
		Data:     string(content),
	}
	if !utf8.Valid(content) {
		payload.Encoding = "base64"
		payload.Data = base64.StdEncoding.EncodeToString(content)
	}

	return renderer.emit(eventType, payload)
}

func (renderer *MachineRenderer) emit(eventType model.EventType, payload any) error {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()

	sequence := renderer.sequence + 1
	event := model.Event{
		Schema:   model.EventSchema,
		Sequence: sequence,
		Command:  renderer.command,
		Type:     eventType,
		Payload:  payload,
	}

	encoded, err := renderer.redactor.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode %s event: %w", eventType, err)
	}
	encoded = append(encoded, '\n')

	written, err := renderer.writer.Write(encoded)
	if err != nil {
		return fmt.Errorf("write %s event: %w", eventType, err)
	}
	if written != len(encoded) {
		return fmt.Errorf("write %s event: %w", eventType, io.ErrShortWrite)
	}

	renderer.sequence = sequence

	return nil
}
