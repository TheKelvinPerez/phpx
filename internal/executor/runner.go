package executor

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
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

type Streams struct {
	Input  io.Reader
	Output io.Writer
	Error  io.Writer
}

type Result struct {
	Started   bool
	ExitCode  int
	Signaled  bool
	Signal    string
	Cancelled bool
}

type StreamRunner interface {
	Run(context.Context, Command, Streams) (Result, error)
}

type OSRunner struct {
	GracePeriod  time.Duration
	SignalSource <-chan os.Signal
}

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
	process.Env = mergeEnvironment(os.Environ(), command.Environment)

	return process.Output()
}

func (runner OSRunner) Run(
	ctx context.Context,
	command Command,
	streams Streams,
) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	process := exec.Command(
		command.Executable,
		command.Arguments...,
	)
	process.Dir = command.WorkingDirectory
	process.Env = mergeEnvironment(os.Environ(), command.Environment)
	process.Stdin = streams.Input
	process.Stdout = streams.Output
	process.Stderr = streams.Error

	if err := process.Start(); err != nil {
		return Result{}, err
	}
	waited := make(chan error, 1)
	go func() {
		waited <- process.Wait()
	}()

	signalSource := runner.SignalSource
	for {
		select {
		case err := <-waited:
			return processResult(process, err, false)
		case <-ctx.Done():
			forwarded := os.Signal(syscall.SIGTERM)
			if received, ok := ContextSignal(ctx); ok {
				forwarded = received
			}
			return runner.stopProcess(
				process,
				waited,
				forwarded,
				true,
			)
		case received, open := <-signalSource:
			if !open {
				signalSource = nil
				continue
			}
			if received == nil {
				continue
			}

			return runner.stopProcess(
				process,
				waited,
				received,
				false,
			)
		}
	}
}

func (runner OSRunner) stopProcess(
	process *exec.Cmd,
	waited <-chan error,
	forwarded os.Signal,
	cancelled bool,
) (Result, error) {
	if err := process.Process.Signal(forwarded); err != nil &&
		!errors.Is(err, os.ErrProcessDone) {
		_ = process.Process.Kill()
		waitErr := <-waited
		result, _ := processResult(process, waitErr, cancelled)

		return result, err
	}

	gracePeriod := runner.GracePeriod
	if gracePeriod <= 0 {
		gracePeriod = 2 * time.Second
	}
	timer := time.NewTimer(gracePeriod)
	defer timer.Stop()
	select {
	case err := <-waited:
		return processResult(process, err, cancelled)
	case <-timer.C:
		if err := process.Process.Kill(); err != nil &&
			!errors.Is(err, os.ErrProcessDone) {
			waitErr := <-waited
			result, _ := processResult(process, waitErr, cancelled)

			return result, err
		}

		return processResult(process, <-waited, cancelled)
	}
}

func processResult(
	process *exec.Cmd,
	err error,
	cancelled bool,
) (Result, error) {
	result := Result{
		Started:   true,
		Cancelled: cancelled,
	}
	if process.ProcessState != nil {
		result.ExitCode = process.ProcessState.ExitCode()
		if waitStatus, ok := process.ProcessState.Sys().(syscall.WaitStatus); ok &&
			waitStatus.Signaled() {
			signal := waitStatus.Signal()
			result.ExitCode = 128 + int(signal)
			result.Signaled = true
			result.Signal = signal.String()
		}
	}
	if err == nil {
		return result, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return result, nil
	}

	return result, err
}

func mergeEnvironment(base []string, overlay []string) []string {
	overridden := make(map[string]int, len(overlay))
	for index, variable := range overlay {
		overridden[environmentName(variable)] = index
	}

	result := make([]string, 0, len(base)+len(overlay))
	for _, variable := range base {
		if _, replaced := overridden[environmentName(variable)]; replaced {
			continue
		}
		result = append(result, variable)
	}
	for index, variable := range overlay {
		if overridden[environmentName(variable)] != index {
			continue
		}
		result = append(result, variable)
	}

	return result
}

func environmentName(variable string) string {
	name, _, found := strings.Cut(variable, "=")
	if !found {
		return variable
	}

	return name
}
