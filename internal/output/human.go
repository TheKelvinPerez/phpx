package output

import (
	"fmt"
	"io"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/security"
)

type HumanRenderer struct {
	stdout   io.Writer
	stderr   io.Writer
	redactor security.Redactor
}

func NewHumanRenderer(stdout io.Writer, stderr io.Writer) *HumanRenderer {
	return NewHumanRendererWithRedactor(
		stdout,
		stderr,
		security.NewRedactor(),
	)
}

func NewHumanRendererWithRedactor(
	stdout io.Writer,
	stderr io.Writer,
	redactor security.Redactor,
) *HumanRenderer {
	return &HumanRenderer{
		stdout:   stdout,
		stderr:   stderr,
		redactor: redactor,
	}
}

func (renderer *HumanRenderer) Started() error {
	return nil
}

func (renderer *HumanRenderer) Fact(fact Fact) error {
	if _, err := fmt.Fprintln(renderer.stdout, renderer.redactor.Text(fact.Text)); err != nil {
		return fmt.Errorf("write human fact: %w", err)
	}

	return nil
}

func (renderer *HumanRenderer) Diagnostic(diagnostic Diagnostic) error {
	if _, err := fmt.Fprintln(renderer.stderr, renderer.redactor.Text(diagnostic.Text)); err != nil {
		return fmt.Errorf("write human diagnostic: %w", err)
	}

	return nil
}

func (renderer *HumanRenderer) Plan(plan Plan) error {
	if _, err := fmt.Fprintln(renderer.stdout, renderer.redactor.Text(plan.Text)); err != nil {
		return fmt.Errorf("write human plan: %w", err)
	}

	return nil
}

func (renderer *HumanRenderer) ApprovalRequired(
	approval ApprovalRequired,
) error {
	if _, err := fmt.Fprintln(
		renderer.stderr,
		renderer.redactor.Text(approval.Text),
	); err != nil {
		return fmt.Errorf("write human approval requirement: %w", err)
	}

	return nil
}

func (renderer *HumanRenderer) Result(result Result) error {
	if _, err := fmt.Fprintln(renderer.stdout, renderer.redactor.Text(result.Text)); err != nil {
		return fmt.Errorf("write human result: %w", err)
	}

	return nil
}

func (renderer *HumanRenderer) Error(commandError *model.Error) error {
	message := renderer.redactor.Text(commandError.Message)
	if _, err := fmt.Fprintf(renderer.stderr, "Error: %s\n", message); err != nil {
		return fmt.Errorf("write human error: %w", err)
	}
	if commandError.Hint == "" {
		return nil
	}
	hint := renderer.redactor.Text(commandError.Hint)
	if _, err := fmt.Fprintf(renderer.stderr, "Hint: %s\n", hint); err != nil {
		return fmt.Errorf("write human hint: %w", err)
	}

	return nil
}

func (renderer *HumanRenderer) Completed(model.Exit) error {
	return nil
}
