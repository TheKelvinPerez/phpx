package executor_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/elefantephp/elefante/internal/executor"
)

func TestOSRunnerPreservesArgumentsDirectoryEnvironmentAndStreams(
	t *testing.T,
) {
	t.Setenv("ELEFANTE_EXECUTOR_OVERLAY", "parent")

	workingDirectory := t.TempDir()
	input := []byte{0x00, 'i', 'n', 'p', 'u', 't', 0xff}
	arguments := []string{
		"plain",
		"space value",
		"$(printf unsafe)",
		"semi;colon",
		"",
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	result, err := (executor.OSRunner{}).Run(
		t.Context(),
		executor.Command{
			Executable: os.Args[0],
			Arguments: append(
				[]string{
					"-test.run=TestExecutorHelperProcess",
					"--",
					"inspect",
				},
				arguments...,
			),
			WorkingDirectory: workingDirectory,
			Environment: []string{
				"ELEFANTE_EXECUTOR_HELPER=1",
				"ELEFANTE_EXECUTOR_OVERLAY=child",
				"ELEFANTE_EXECUTOR_ADDED=added",
			},
		},
		executor.Streams{
			Input:  bytes.NewReader(input),
			Output: &stdout,
			Error:  &stderr,
		},
	)
	if err != nil {
		t.Fatalf("run helper process: %v", err)
	}
	if !result.Started || result.ExitCode != 0 ||
		result.Signaled || result.Signal != "" ||
		result.Cancelled {
		t.Fatalf("unexpected successful process result %#v", result)
	}

	var observed helperObservation
	if err := json.Unmarshal(stdout.Bytes(), &observed); err != nil {
		t.Fatalf("decode helper observation: %v\nstdout:\n%s", err, stdout.String())
	}
	resolvedWorkingDirectory, err := filepath.EvalSymlinks(workingDirectory)
	if err != nil {
		t.Fatalf("resolve expected working directory: %v", err)
	}
	resolvedObservedDirectory, err := filepath.EvalSymlinks(observed.WorkingDirectory)
	if err != nil {
		t.Fatalf("resolve observed working directory: %v", err)
	}
	if resolvedObservedDirectory != resolvedWorkingDirectory {
		t.Fatalf(
			"working directory changed from %q to %q",
			resolvedWorkingDirectory,
			resolvedObservedDirectory,
		)
	}
	if !slices.Equal(observed.Arguments, arguments) {
		t.Fatalf(
			"argument vector changed\nexpected: %#v\ngot:      %#v",
			arguments,
			observed.Arguments,
		)
	}
	if !bytes.Equal(observed.Input, input) {
		t.Fatalf(
			"standard input changed\nexpected: %v\ngot:      %v",
			input,
			observed.Input,
		)
	}
	if observed.Overlay != "child" || observed.Added != "added" {
		t.Fatalf("environment overlay lost precedence %#v", observed)
	}
	if stderr.String() != "helper-stderr\n" {
		t.Fatalf("standard error changed: %q", stderr.String())
	}
}

func TestOSRunnerPreservesNonzeroExitCodeAfterStart(t *testing.T) {
	result, err := (executor.OSRunner{}).Run(
		t.Context(),
		executor.Command{
			Executable: os.Args[0],
			Arguments: []string{
				"-test.run=TestExecutorHelperProcess",
				"--",
				"exit",
				"37",
			},
			Environment: []string{"ELEFANTE_EXECUTOR_HELPER=1"},
		},
		executor.Streams{},
	)
	if err != nil {
		t.Fatalf("wait for nonzero helper process: %v", err)
	}
	if !result.Started || result.ExitCode != 37 ||
		result.Signaled || result.Signal != "" ||
		result.Cancelled {
		t.Fatalf("nonzero child exit was not preserved %#v", result)
	}
}

func TestOSRunnerDistinguishesStartAndStreamFailures(t *testing.T) {
	t.Run("start failure", func(t *testing.T) {
		result, err := (executor.OSRunner{}).Run(
			t.Context(),
			executor.Command{
				Executable: filepath.Join(t.TempDir(), "missing"),
			},
			executor.Streams{},
		)
		if err == nil {
			t.Fatal("expected missing executable start failure")
		}
		if result.Started {
			t.Fatalf("start failure reported a started child %#v", result)
		}
	})

	t.Run("stream failure after start", func(t *testing.T) {
		streamFailure := errors.New("synthetic stream failure")
		result, err := (executor.OSRunner{}).Run(
			t.Context(),
			executor.Command{
				Executable: os.Args[0],
				Arguments: []string{
					"-test.run=TestExecutorHelperProcess",
					"--",
					"write",
				},
				Environment: []string{"ELEFANTE_EXECUTOR_HELPER=1"},
			},
			executor.Streams{
				Output: failingWriter{err: streamFailure},
			},
		)
		if !errors.Is(err, streamFailure) {
			t.Fatalf("expected stream failure cause, got %v", err)
		}
		if !result.Started || result.ExitCode != 0 {
			t.Fatalf("stream failure lost child result %#v", result)
		}
	})
}

func TestOSRunnerCancellationForwardsTerminationBeforeForcedKill(
	t *testing.T,
) {
	ctx, cancel := context.WithCancel(t.Context())
	output := newNotifyingBuffer("helper-ready\n")
	type outcome struct {
		result executor.Result
		err    error
	}
	finished := make(chan outcome, 1)

	go func() {
		result, err := (executor.OSRunner{
			GracePeriod: time.Second,
		}).Run(
			ctx,
			executor.Command{
				Executable: os.Args[0],
				Arguments: []string{
					"-test.run=TestExecutorHelperProcess",
					"--",
					"wait-for-termination",
				},
				Environment: []string{"ELEFANTE_EXECUTOR_HELPER=1"},
			},
			executor.Streams{Output: output},
		)
		finished <- outcome{result: result, err: err}
	}()

	select {
	case <-output.Notified():
	case <-time.After(5 * time.Second):
		t.Fatal("helper process did not become ready")
	}
	cancel()

	var observed outcome
	select {
	case observed = <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled helper process did not exit")
	}
	if observed.err != nil {
		t.Fatalf("cancel helper process: %v", observed.err)
	}
	if !observed.result.Started ||
		observed.result.ExitCode != 42 ||
		observed.result.Signaled ||
		observed.result.Signal != "" ||
		!observed.result.Cancelled {
		t.Fatalf(
			"graceful cancellation result was not preserved %#v",
			observed.result,
		)
	}
	if !strings.Contains(
		output.String(),
		"received-signal=terminated\n",
	) {
		t.Fatalf(
			"child did not observe forwarded termination signal:\n%s",
			output.String(),
		)
	}
}

func TestOSRunnerCancellationForcesKillAfterGracePeriod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	output := newNotifyingBuffer("helper-ready\n")
	type outcome struct {
		result executor.Result
		err    error
	}
	finished := make(chan outcome, 1)

	go func() {
		result, err := (executor.OSRunner{
			GracePeriod: 50 * time.Millisecond,
		}).Run(
			ctx,
			executor.Command{
				Executable: os.Args[0],
				Arguments: []string{
					"-test.run=TestExecutorHelperProcess",
					"--",
					"ignore-termination",
				},
				Environment: []string{"ELEFANTE_EXECUTOR_HELPER=1"},
			},
			executor.Streams{Output: output},
		)
		finished <- outcome{result: result, err: err}
	}()

	select {
	case <-output.Notified():
	case <-time.After(5 * time.Second):
		t.Fatal("signal ignoring helper process did not become ready")
	}
	cancel()

	var observed outcome
	select {
	case observed = <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("signal ignoring helper process was not killed")
	}
	if observed.err != nil {
		t.Fatalf("force helper process termination: %v", observed.err)
	}
	if !observed.result.Started ||
		observed.result.ExitCode != 137 ||
		!observed.result.Signaled ||
		observed.result.Signal != "killed" ||
		!observed.result.Cancelled {
		t.Fatalf(
			"forced cancellation result was not preserved %#v",
			observed.result,
		)
	}
}

func TestOSRunnerForwardsSupportedSignals(t *testing.T) {
	signals := make(chan os.Signal, 1)
	output := newNotifyingBuffer("helper-ready\n")
	type outcome struct {
		result executor.Result
		err    error
	}
	finished := make(chan outcome, 1)

	go func() {
		result, err := (executor.OSRunner{
			GracePeriod:  time.Second,
			SignalSource: signals,
		}).Run(
			t.Context(),
			executor.Command{
				Executable: os.Args[0],
				Arguments: []string{
					"-test.run=TestExecutorHelperProcess",
					"--",
					"wait-for-interrupt",
				},
				Environment: []string{"ELEFANTE_EXECUTOR_HELPER=1"},
			},
			executor.Streams{Output: output},
		)
		finished <- outcome{result: result, err: err}
	}()

	select {
	case <-output.Notified():
	case <-time.After(5 * time.Second):
		t.Fatal("signal forwarding helper process did not become ready")
	}
	signals <- os.Interrupt

	var observed outcome
	select {
	case observed = <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("signal forwarding helper process did not exit")
	}
	if observed.err != nil {
		t.Fatalf("forward helper signal: %v", observed.err)
	}
	if !observed.result.Started ||
		observed.result.ExitCode != 43 ||
		observed.result.Signaled ||
		observed.result.Signal != "" ||
		observed.result.Cancelled {
		t.Fatalf(
			"signal handled child result was not preserved %#v",
			observed.result,
		)
	}
	if !strings.Contains(
		output.String(),
		"received-signal=interrupt\n",
	) {
		t.Fatalf(
			"child did not observe forwarded interrupt:\n%s",
			output.String(),
		)
	}
}

type helperObservation struct {
	Arguments        []string `json:"arguments"`
	WorkingDirectory string   `json:"working_directory"`
	Overlay          string   `json:"overlay"`
	Added            string   `json:"added"`
	Input            []byte   `json:"input"`
}

func TestExecutorHelperProcess(t *testing.T) {
	if os.Getenv("ELEFANTE_EXECUTOR_HELPER") != "1" {
		return
	}

	separator := slices.Index(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) {
		fmt.Fprintln(os.Stderr, "missing helper command")
		os.Exit(64)
	}
	command := os.Args[separator+1]
	arguments := os.Args[separator+2:]

	switch command {
	case "inspect":
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read helper input: %v\n", err)
			os.Exit(70)
		}
		workingDirectory, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "read helper working directory: %v\n", err)
			os.Exit(70)
		}
		observed := helperObservation{
			Arguments:        append([]string(nil), arguments...),
			WorkingDirectory: workingDirectory,
			Overlay:          os.Getenv("ELEFANTE_EXECUTOR_OVERLAY"),
			Added:            os.Getenv("ELEFANTE_EXECUTOR_ADDED"),
			Input:            input,
		}
		if err := json.NewEncoder(os.Stdout).Encode(observed); err != nil {
			fmt.Fprintf(os.Stderr, "encode helper observation: %v\n", err)
			os.Exit(70)
		}
		fmt.Fprintln(os.Stderr, "helper-stderr")
		os.Exit(0)
	case "exit":
		if len(arguments) != 1 {
			fmt.Fprintln(os.Stderr, "exit helper requires one code")
			os.Exit(64)
		}
		code, err := strconv.Atoi(arguments[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse helper exit code: %v\n", err)
			os.Exit(64)
		}
		os.Exit(code)
	case "write":
		fmt.Fprintln(os.Stdout, "helper-output")
		os.Exit(0)
	case "wait-for-termination":
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM)
		fmt.Fprintln(os.Stdout, "helper-ready")
		received := <-signals
		signal.Stop(signals)
		fmt.Fprintf(os.Stdout, "received-signal=%s\n", received)
		os.Exit(42)
	case "ignore-termination":
		signal.Ignore(syscall.SIGTERM)
		fmt.Fprintln(os.Stdout, "helper-ready")
		for {
			time.Sleep(time.Second)
		}
	case "wait-for-interrupt":
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		fmt.Fprintln(os.Stdout, "helper-ready")
		received := <-signals
		signal.Stop(signals)
		fmt.Fprintf(os.Stdout, "received-signal=%s\n", received)
		os.Exit(43)
	default:
		fmt.Fprintf(
			os.Stderr,
			"unknown helper command %q with %s\n",
			command,
			strings.Join(arguments, ","),
		)
		os.Exit(64)
	}
}

type failingWriter struct {
	err error
}

func (writer failingWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

type notifyingBuffer struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	pattern  string
	notified chan struct{}
	once     sync.Once
}

func newNotifyingBuffer(pattern string) *notifyingBuffer {
	return &notifyingBuffer{
		pattern:  pattern,
		notified: make(chan struct{}),
	}
}

func (buffer *notifyingBuffer) Write(content []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	written, err := buffer.buffer.Write(content)
	if strings.Contains(buffer.buffer.String(), buffer.pattern) {
		buffer.once.Do(func() {
			close(buffer.notified)
		})
	}

	return written, err
}

func (buffer *notifyingBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	return buffer.buffer.String()
}

func (buffer *notifyingBuffer) Notified() <-chan struct{} {
	return buffer.notified
}
