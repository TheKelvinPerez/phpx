package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/elefantephp/elefante/internal/model"
)

type MachineRenderer struct {
	mu       sync.Mutex
	writer   io.Writer
	command  string
	sequence uint64
}

func NewMachineRenderer(writer io.Writer, command string) *MachineRenderer {
	return &MachineRenderer{
		writer:  writer,
		command: command,
	}
}

func (renderer *MachineRenderer) Started() error {
	return renderer.emit(model.EventStarted, struct{}{})
}

func (renderer *MachineRenderer) Fact(fact Fact) error {
	return renderer.emit(model.EventFact, fact.Payload)
}

func (renderer *MachineRenderer) Result(result Result) error {
	return renderer.emit(model.EventResult, result.Payload)
}

func (renderer *MachineRenderer) Error(commandError *model.Error) error {
	return renderer.emit(model.EventError, commandError)
}

func (renderer *MachineRenderer) Completed(exit model.Exit) error {
	return renderer.emit(model.EventCompleted, model.CompletedPayload{
		Exit: exit,
	})
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

	encoded, err := json.Marshal(event)
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
