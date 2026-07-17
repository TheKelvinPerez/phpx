package main_test

import (
	"bytes"
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

func buildBinary(t *testing.T) string {
	t.Helper()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	repositoryRoot := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	binary := filepath.Join(t.TempDir(), "elefante")
	command := exec.Command("go", "build", "-o", binary, "./cmd/elefante")
	command.Dir = repositoryRoot

	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build elefante: %v\noutput:\n%s", err, output)
	}

	return binary
}
