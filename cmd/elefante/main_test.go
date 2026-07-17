package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/model"
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

func TestCompiledBinaryJSONDoctorDiscoversProjectFromDescendant(t *testing.T) {
	binary := buildBinary(t)
	projectRoot := t.TempDir()
	composerContent := "{\"name\":\"acme/example\"}\n"
	if err := os.WriteFile(
		filepath.Join(projectRoot, "composer.json"),
		[]byte(composerContent),
		0o644,
	); err != nil {
		t.Fatalf("write Composer fixture: %v", err)
	}
	descendant := filepath.Join(projectRoot, "src", "Domain")
	if err := os.MkdirAll(descendant, 0o755); err != nil {
		t.Fatalf("create project descendant: %v", err)
	}

	first := runCompiledDoctor(t, binary, descendant)
	second := runCompiledDoctor(t, binary, descendant)
	if first != second {
		t.Fatalf(
			"expected deterministic doctor events\nfirst:\n%s\nsecond:\n%s",
			first,
			second,
		)
	}

	events := decodeCompiledEvents(t, first)
	if len(events) != 3 {
		t.Fatalf("expected started, fact, and completed events, got %d", len(events))
	}
	expectedTypes := []model.EventType{
		model.EventStarted,
		model.EventFact,
		model.EventCompleted,
	}
	for index, expectedType := range expectedTypes {
		if events[index].Schema != model.EventSchema {
			t.Errorf("event %d has unexpected schema %q", index+1, events[index].Schema)
		}
		if events[index].Sequence != uint64(index+1) {
			t.Errorf("event %d has sequence %d", index+1, events[index].Sequence)
		}
		if events[index].Command != "doctor" {
			t.Errorf("event %d has command %q", index+1, events[index].Command)
		}
		if events[index].Type != expectedType {
			t.Errorf("event %d has type %q", index+1, events[index].Type)
		}
	}

	var facts model.ProjectFacts
	if err := json.Unmarshal(events[1].Payload, &facts); err != nil {
		t.Fatalf("decode doctor facts: %v", err)
	}
	resolvedProjectRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	if facts.Identity.ComposerRoot != resolvedProjectRoot {
		t.Errorf(
			"expected Composer root %q, got %q",
			resolvedProjectRoot,
			facts.Identity.ComposerRoot,
		)
	}
	if facts.Identity.WorkspaceRoot != resolvedProjectRoot {
		t.Errorf(
			"expected workspace root %q, got %q",
			resolvedProjectRoot,
			facts.Identity.WorkspaceRoot,
		)
	}
	if len(facts.InputFingerprints) != 1 {
		t.Fatalf("expected one discovery input fingerprint, got %#v", facts.InputFingerprints)
	}
}

func TestCompiledBinaryJSONDoctorDiscoversGitRepositoryFromCurrentDirectory(t *testing.T) {
	binary := buildBinary(t)
	projectRoot := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(projectRoot, "composer.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatalf("write Composer fixture: %v", err)
	}
	runGitFixture(t, projectRoot, "init", "-b", "main")
	runGitFixture(t, projectRoot, "config", "user.name", "Elefante Tests")
	runGitFixture(t, projectRoot, "config", "user.email", "tests@elefante.local")
	runGitFixture(t, projectRoot, "add", "composer.json")
	runGitFixture(t, projectRoot, "commit", "-m", "Initial fixture")

	output := runCompiledDoctorFromDirectory(t, binary, projectRoot)
	facts := decodeDoctorFacts(t, output)

	resolvedProjectRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	if facts.Identity.RepositoryRoot != resolvedProjectRoot {
		t.Errorf(
			"expected repository root %q, got %q",
			resolvedProjectRoot,
			facts.Identity.RepositoryRoot,
		)
	}
	if facts.Identity.WorkspaceRoot != resolvedProjectRoot {
		t.Errorf(
			"expected workspace root %q, got %q",
			resolvedProjectRoot,
			facts.Identity.WorkspaceRoot,
		)
	}
	if facts.Identity.Branch != "main" {
		t.Errorf("expected branch main, got %q", facts.Identity.Branch)
	}
}

func TestCompiledBinaryJSONDoctorDistinguishesLinkedWorktree(t *testing.T) {
	binary := buildBinary(t)
	repositoryRoot := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(repositoryRoot, "composer.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatalf("write Composer fixture: %v", err)
	}
	runGitFixture(t, repositoryRoot, "init", "-b", "main")
	runGitFixture(t, repositoryRoot, "config", "user.name", "Elefante Tests")
	runGitFixture(t, repositoryRoot, "config", "user.email", "tests@elefante.local")
	runGitFixture(t, repositoryRoot, "add", "composer.json")
	runGitFixture(t, repositoryRoot, "commit", "-m", "Initial fixture")

	worktreeRoot := filepath.Join(t.TempDir(), "feature")
	runGitFixture(t, repositoryRoot, "worktree", "add", "-b", "feature", worktreeRoot)

	mainFacts := decodeDoctorFacts(t, runCompiledDoctor(t, binary, repositoryRoot))
	worktreeFacts := decodeDoctorFacts(t, runCompiledDoctor(t, binary, worktreeRoot))

	if mainFacts.Identity.GitCommonDir != worktreeFacts.Identity.GitCommonDir {
		t.Errorf(
			"expected shared Git common directory, got %q and %q",
			mainFacts.Identity.GitCommonDir,
			worktreeFacts.Identity.GitCommonDir,
		)
	}
	if mainFacts.Identity.IdentityKey == worktreeFacts.Identity.IdentityKey {
		t.Errorf("expected distinct worktree identity, got %q", mainFacts.Identity.IdentityKey)
	}
	if worktreeFacts.Identity.Branch != "feature" {
		t.Errorf("expected worktree branch feature, got %q", worktreeFacts.Identity.Branch)
	}
}

func TestCompiledBinaryJSONDoctorReportsAmbiguousRepository(t *testing.T) {
	binary := buildBinary(t)
	repositoryRoot := t.TempDir()
	runGitFixture(t, repositoryRoot, "init", "-b", "main")

	firstRoot := filepath.Join(repositoryRoot, "apps", "api")
	secondRoot := filepath.Join(repositoryRoot, "apps", "worker")
	for _, root := range []string{firstRoot, secondRoot} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("create Composer root: %v", err)
		}
		if err := os.WriteFile(
			filepath.Join(root, "composer.json"),
			[]byte("{}\n"),
			0o644,
		); err != nil {
			t.Fatalf("write Composer fixture: %v", err)
		}
	}

	command := exec.Command(binary, "--json", "--project", repositoryRoot, "doctor")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("expected discovery process error, got %v", err)
	}
	if exitError.ExitCode() != 3 {
		t.Fatalf("expected discovery exit 3, got %d", exitError.ExitCode())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected JSON mode stderr to be empty, got:\n%s", stderr.String())
	}

	events := decodeCompiledEvents(t, stdout.String())
	if len(events) != 3 {
		t.Fatalf("expected started, error, and completed events, got %d", len(events))
	}
	if events[0].Type != model.EventStarted ||
		events[1].Type != model.EventError ||
		events[2].Type != model.EventCompleted {
		t.Fatalf(
			"unexpected ambiguity event sequence %q, %q, %q",
			events[0].Type,
			events[1].Type,
			events[2].Type,
		)
	}

	var commandError model.Error
	if err := json.Unmarshal(events[1].Payload, &commandError); err != nil {
		t.Fatalf("decode discovery error: %v", err)
	}
	if commandError.Code != model.ErrorDiscoveryAmbiguousRoots {
		t.Errorf("expected ambiguity code, got %q", commandError.Code)
	}
	if len(commandError.Details) != 2 {
		t.Fatalf("expected two ambiguity candidates, got %#v", commandError.Details)
	}
}

type compiledEvent struct {
	Schema   string          `json:"schema"`
	Sequence uint64          `json:"sequence"`
	Command  string          `json:"command"`
	Type     model.EventType `json:"type"`
	Payload  json.RawMessage `json:"payload"`
}

func runCompiledDoctor(t *testing.T, binary string, projectPath string) string {
	t.Helper()

	command := exec.Command(binary, "--json", "--project", projectPath, "doctor")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("run elefante doctor: %v\nstderr:\n%s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected doctor stderr to be empty, got:\n%s", stderr.String())
	}

	return stdout.String()
}

func runCompiledDoctorFromDirectory(t *testing.T, binary string, directory string) string {
	t.Helper()

	command := exec.Command(binary, "--json", "doctor")
	command.Dir = directory
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("run elefante doctor: %v\nstderr:\n%s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected doctor stderr to be empty, got:\n%s", stderr.String())
	}

	return stdout.String()
}

func decodeDoctorFacts(t *testing.T, content string) model.ProjectFacts {
	t.Helper()

	events := decodeCompiledEvents(t, content)
	if len(events) != 3 || events[1].Type != model.EventFact {
		t.Fatalf("expected doctor fact event, got %#v", events)
	}

	var facts model.ProjectFacts
	if err := json.Unmarshal(events[1].Payload, &facts); err != nil {
		t.Fatalf("decode doctor facts: %v", err)
	}

	return facts
}

func decodeCompiledEvents(t *testing.T, content string) []compiledEvent {
	t.Helper()

	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	events := make([]compiledEvent, 0, len(lines))
	for index, line := range lines {
		var event compiledEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode event line %d: %v\nline:\n%s", index+1, err, line)
		}
		events = append(events, event)
	}

	return events
}

func runGitFixture(t *testing.T, directory string, arguments ...string) string {
	t.Helper()

	command := exec.Command("git", arguments...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\noutput:\n%s", strings.Join(arguments, " "), err, output)
	}

	return strings.TrimSpace(string(output))
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
