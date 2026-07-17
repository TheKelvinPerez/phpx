package main_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompiledBinaryHelp(t *testing.T) {
	binary := buildBinary(t)

	command := exec.Command(binary, "--help")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("run elefante --help: %v\nstderr:\n%s", err, stderr.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got:\n%s", stderr.String())
	}

	for _, expected := range []string{
		"The local development runtime for PHP.",
		"Usage:",
		"elefante",
	} {
		if !strings.Contains(stdout.String(), expected) {
			t.Errorf("expected help output to contain %q, got:\n%s", expected, stdout.String())
		}
	}
}

func TestCompiledBinaryVersion(t *testing.T) {
	binary := buildBinary(t)

	command := exec.Command(binary, "version")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("run elefante version: %v\nstderr:\n%s", err, stderr.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got:\n%s", stderr.String())
	}

	if got, expected := stdout.String(), "elefante dev\n"; got != expected {
		t.Fatalf("expected version output %q, got %q", expected, got)
	}
}

func TestCompiledBinaryUsageError(t *testing.T) {
	binary := buildBinary(t)

	command := exec.Command(binary, "unknown")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("expected a process exit error, got %v", err)
	}
	if exitError.ExitCode() != 2 {
		t.Fatalf("expected usage exit 2, got %d", exitError.ExitCode())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got:\n%s", stdout.String())
	}

	expected := "Error: unknown command \"unknown\" for \"elefante\"\nHint: Run elefante --help to see available commands.\n"
	if stderr.String() != expected {
		t.Fatalf("expected human error %q, got %q", expected, stderr.String())
	}
}

func TestCompiledBinaryJSONVersion(t *testing.T) {
	binary := buildBinary(t)

	command := exec.Command(binary, "--json", "version")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("run elefante --json version: %v\nstderr:\n%s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got:\n%s", stderr.String())
	}

	expected := readEventGolden(t, "version-success.ndjson")
	if stdout.String() != expected {
		t.Fatalf("compiled JSON output does not match golden\nexpected:\n%s\ngot:\n%s", expected, stdout.String())
	}
}

func TestCompiledBinaryJSONUsageError(t *testing.T) {
	binary := buildBinary(t)

	command := exec.Command(binary, "--json", "unknown")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("expected a process exit error, got %v", err)
	}
	if exitError.ExitCode() != 2 {
		t.Fatalf("expected usage exit 2, got %d", exitError.ExitCode())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected JSON mode stderr to be empty, got:\n%s", stderr.String())
	}

	expected := readEventGolden(t, "usage-error.ndjson")
	if stdout.String() != expected {
		t.Fatalf("compiled JSON error does not match golden\nexpected:\n%s\ngot:\n%s", expected, stdout.String())
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()

	repositoryRoot := repositoryRoot(t)
	binary := filepath.Join(t.TempDir(), "elefante")
	command := exec.Command("go", "build", "-o", binary, "./cmd/elefante")
	command.Dir = repositoryRoot

	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build elefante: %v\noutput:\n%s", err, output)
	}

	return binary
}

func readEventGolden(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(repositoryRoot(t), "testdata", "golden", "events", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}

	return string(content)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	return filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
}
