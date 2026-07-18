package cli

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/output"
)

type Execution struct {
	Arguments []string
	Input     io.Reader
	Output    io.Writer
	Error     io.Writer
}

func Execute(ctx context.Context, dependencies Dependencies, execution Execution) int {
	commandName := selectedCommand(execution.Arguments)
	renderer := newRenderer(execution, commandName)
	dependencies.Renderer = renderer

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
	if err := renderer.Completed(exit); err != nil {
		return model.ExitCode(model.WrapError(
			model.ErrorInternal,
			"Could not write the command completion event.",
			err,
		))
	}

	return 0
}

func newRenderer(execution Execution, commandName string) output.Renderer {
	if requestsJSON(execution.Arguments) {
		return output.NewMachineRenderer(execution.Output, commandName)
	}

	return output.NewHumanRenderer(execution.Output, execution.Error)
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
			argument == "--provider" {
			index++
			continue
		}
		if strings.HasPrefix(argument, "--project=") ||
			strings.HasPrefix(argument, "--config=") ||
			strings.HasPrefix(argument, "--provider=") {
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
