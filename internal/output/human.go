package output

import (
	"fmt"
	"io"

	"github.com/elefantephp/elefante/internal/model"
)

type HumanRenderer struct {
	stdout io.Writer
	stderr io.Writer
}

func NewHumanRenderer(stdout io.Writer, stderr io.Writer) *HumanRenderer {
	return &HumanRenderer{
		stdout: stdout,
		stderr: stderr,
	}
}

func (renderer *HumanRenderer) Started() error {
	return nil
}

func (renderer *HumanRenderer) Fact(fact Fact) error {
	if _, err := fmt.Fprintln(renderer.stdout, fact.Text); err != nil {
		return fmt.Errorf("write human fact: %w", err)
	}

	return nil
}

func (renderer *HumanRenderer) Result(result Result) error {
	if _, err := fmt.Fprintln(renderer.stdout, result.Text); err != nil {
		return fmt.Errorf("write human result: %w", err)
	}

	return nil
}

func (renderer *HumanRenderer) Error(commandError *model.Error) error {
	if _, err := fmt.Fprintf(renderer.stderr, "Error: %s\n", commandError.Message); err != nil {
		return fmt.Errorf("write human error: %w", err)
	}
	if commandError.Hint == "" {
		return nil
	}
	if _, err := fmt.Fprintf(renderer.stderr, "Hint: %s\n", commandError.Hint); err != nil {
		return fmt.Errorf("write human hint: %w", err)
	}

	return nil
}

func (renderer *HumanRenderer) Completed(model.Exit) error {
	return nil
}
