package executor

import (
	"context"
	"os"
	"os/exec"
)

type Command struct {
	Executable       string
	Arguments        []string
	WorkingDirectory string
	Environment      []string
}

type Runner interface {
	LookPath(string) (string, error)
	Output(context.Context, Command) ([]byte, error)
}

type OSRunner struct{}

func (OSRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (OSRunner) Output(
	ctx context.Context,
	command Command,
) ([]byte, error) {
	process := exec.CommandContext(
		ctx,
		command.Executable,
		command.Arguments...,
	)
	process.Dir = command.WorkingDirectory
	process.Env = append(os.Environ(), command.Environment...)

	return process.Output()
}
