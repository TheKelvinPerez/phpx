package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/output"
	"github.com/elefantephp/elefante/internal/security"
)

type Execution struct {
	Arguments       []string
	Environment     []string
	Input           io.Reader
	InputIsTerminal bool
	Output          io.Writer
	Error           io.Writer
}

func Execute(ctx context.Context, dependencies Dependencies, execution Execution) int {
	commandName := selectedCommand(execution.Arguments)
	renderer := newRenderer(execution, commandName)
	dependencies.Renderer = renderer
	state := &commandState{}
	dependencies.state = state
	dependencies.childOutput = execution.Output
	dependencies.childError = execution.Error
	if streamRenderer, ok := renderer.(childStreamRenderer); ok &&
		requestsJSON(execution.Arguments) {
		dependencies.childOutput = rendererStreamWriter{
			write: streamRenderer.Stdout,
		}
		dependencies.childError = rendererStreamWriter{
			write: streamRenderer.Stderr,
		}
	}
	dependencies.AllowPrompts = !requestsJSON(execution.Arguments) &&
		(execution.InputIsTerminal || terminalInput(execution.Input))

	if err := renderer.Started(); err != nil {
		return model.ExitCode(model.WrapError(
			model.ErrorInternal,
			"Could not write the command start event.",
			err,
		))
	}

	root := NewRootCommand(dependencies)
	root.SetArgs(execution.Arguments)
	root.SetIn(execution.Input)
	if requestsJSON(execution.Arguments) {
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
	} else {
		root.SetOut(execution.Output)
		root.SetErr(execution.Error)
	}

	if err := root.ExecuteContext(ctx); err != nil {
		commandError := publicCommandError(err)
		if renderErr := renderer.Error(commandError); renderErr != nil {
			return model.ExitCode(model.WrapError(
				model.ErrorInternal,
				"Could not write the command error.",
				renderErr,
			))
		}

		exit := model.ExitForError(commandError)
		if state.exit != nil {
			exit = *state.exit
		}
		if renderErr := renderer.Completed(exit); renderErr != nil {
			return model.ExitCode(model.WrapError(
				model.ErrorInternal,
				"Could not write the command completion event.",
				renderErr,
			))
		}

		return exit.Code
	}

	exit := model.Exit{
		Origin: model.ExitOriginElefante,
		Code:   0,
	}
	if state.exit != nil {
		exit = *state.exit
	}
	if err := renderer.Completed(exit); err != nil {
		return model.ExitCode(model.WrapError(
			model.ErrorInternal,
			"Could not write the command completion event.",
			err,
		))
	}

	return exit.Code
}

type childStreamRenderer interface {
	Stdout([]byte) error
	Stderr([]byte) error
}

type rendererStreamWriter struct {
	write func([]byte) error
}

func (writer rendererStreamWriter) Write(content []byte) (int, error) {
	if len(content) == 0 {
		return 0, nil
	}
	cloned := append([]byte(nil), content...)
	if err := writer.write(cloned); err != nil {
		return 0, err
	}

	return len(content), nil
}

func terminalInput(input io.Reader) bool {
	file, ok := input.(*os.File)
	if !ok || file.Name() == os.DevNull {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func newRenderer(execution Execution, commandName string) output.Renderer {
	redactor := security.NewEnvironmentRedactor(execution.Environment)
	if requestsJSON(execution.Arguments) {
		return output.NewMachineRendererWithRedactor(
			execution.Output,
			commandName,
			redactor,
		)
	}

	return output.NewHumanRendererWithRedactor(
		execution.Output,
		execution.Error,
		redactor,
	)
}

func requestsJSON(arguments []string) bool {
	for _, argument := range arguments {
		if argument == "--" {
			break
		}
		if argument == "--json" {
			return true
		}
		if !strings.HasPrefix(argument, "--json=") {
			continue
		}

		value := strings.TrimPrefix(argument, "--json=")
		enabled, err := strconv.ParseBool(value)

		return err != nil || enabled
	}

	return false
}

func selectedCommand(arguments []string) string {
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		if argument == "--" {
			break
		}
		if argument == "--project" ||
			argument == "--config" ||
			argument == "--provider" ||
			argument == "--approve-plan" {
			index++
			continue
		}
		if strings.HasPrefix(argument, "--project=") ||
			strings.HasPrefix(argument, "--config=") ||
			strings.HasPrefix(argument, "--provider=") ||
			strings.HasPrefix(argument, "--approve-plan=") {
			continue
		}
		if strings.HasPrefix(argument, "-") {
			continue
		}

		return argument
	}

	return "elefante"
}

func publicCommandError(err error) *model.Error {
	var commandError *model.Error
	if errors.As(err, &commandError) {
		return commandError
	}

	return model.WrapError(model.ErrorUsage, err.Error(), err).
		WithHint("Run elefante --help to see available commands.")
}
