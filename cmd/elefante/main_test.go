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
	if len(events) < 5 {
		t.Fatalf("expected doctor analysis events, got %d", len(events))
	}
	if events[0].Type != model.EventStarted {
		t.Fatalf("expected started event first, got %q", events[0].Type)
	}
	if events[len(events)-1].Type != model.EventCompleted {
		t.Fatalf(
			"expected completed event last, got %q",
			events[len(events)-1].Type,
		)
	}
	foundPlan := false
	for index := range events {
		if events[index].Schema != model.EventSchema {
			t.Errorf("event %d has unexpected schema %q", index+1, events[index].Schema)
		}
		if events[index].Sequence != uint64(index+1) {
			t.Errorf("event %d has sequence %d", index+1, events[index].Sequence)
		}
		if events[index].Command != "doctor" {
			t.Errorf("event %d has command %q", index+1, events[index].Command)
		}
		if events[index].Type == model.EventPlan {
			foundPlan = true
		}
	}
	if !foundPlan {
		t.Fatal("expected doctor plan event")
	}

	facts := projectFactsFromEvents(t, events)
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

func TestCompiledBinaryNativeDoctorAndPlanInspectLocalExecutables(t *testing.T) {
	if _, err := exec.LookPath("php"); err != nil {
		t.Skip("local PHP executable is unavailable")
	}
	if _, err := exec.LookPath("composer"); err != nil {
		t.Skip("local Composer executable is unavailable")
	}

	binary := buildBinary(t)
	projectRoot := t.TempDir()
	composerContent := `{
    "name": "acme/native-proof",
    "require": {
        "php": ">=8.0",
        "ext-json": "*"
    }
}
`
	if err := os.WriteFile(
		filepath.Join(projectRoot, "composer.json"),
		[]byte(composerContent),
		0o644,
	); err != nil {
		t.Fatalf("write native Composer fixture: %v", err)
	}

	doctorOutput := runCompiledAnalysis(
		t,
		binary,
		"--json",
		"--project",
		projectRoot,
		"--provider",
		"native",
		"doctor",
	)
	doctorEvents := decodeCompiledEvents(t, doctorOutput)
	observation := providerObservationFromEvents(t, doctorEvents, "native")
	if len(observation.Runtimes) != 1 ||
		observation.Runtimes[0].Name != "php" ||
		observation.Runtimes[0].Version == "" ||
		observation.Runtimes[0].SAPI == "" ||
		observation.Runtimes[0].Source.Path == "" {
		t.Fatalf("unexpected local PHP observation %#v", observation.Runtimes)
	}
	if len(observation.Composer) != 1 ||
		observation.Composer[0].Version == "" ||
		observation.Composer[0].Path == "" ||
		observation.Composer[0].Identity == "" {
		t.Fatalf("unexpected local Composer observation %#v", observation.Composer)
	}
	foundJSON := false
	for _, extension := range observation.Extensions {
		if extension.Name == "ext-json" && extension.Available {
			foundJSON = true
			if extension.Source.Path == "" {
				t.Fatalf("expected extension provenance, got %#v", extension)
			}
		}
	}
	if !foundJSON {
		t.Fatalf("expected local ext-json observation, got %#v", observation.Extensions)
	}
	doctorPlan := planFromEvents(t, doctorEvents)
	if doctorPlan.Operation != model.OperationDoctor ||
		doctorPlan.Provider.Name != "native" ||
		doctorPlan.Provider.Reason != "explicit" ||
		len(doctorPlan.Actions) != 0 {
		t.Fatalf("unexpected native doctor plan %#v", doctorPlan)
	}

	planOutput := runCompiledAnalysis(
		t,
		binary,
		"--json",
		"--project",
		projectRoot,
		"--provider",
		"native",
		"plan",
	)
	syncPlan := planFromEvents(t, decodeCompiledEvents(t, planOutput))
	if syncPlan.Operation != model.OperationSync ||
		syncPlan.Provider.Name != "native" ||
		!strings.HasPrefix(syncPlan.Digest, "sha256:") {
		t.Fatalf("unexpected native synchronization plan %#v", syncPlan)
	}
	for _, action := range syncPlan.Actions {
		if action.Kind == model.ActionPrepareRuntime ||
			action.Kind == model.ActionPrepareExtension {
			t.Fatalf(
				"compatible native plan must not install or relink PHP: %#v",
				action,
			)
		}
	}

	for _, commandName := range []string{"doctor", "plan"} {
		stdout, stderr := runCompiledHumanAnalysis(
			t,
			binary,
			"--project",
			projectRoot,
			"--provider",
			"native",
			commandName,
		)
		expectedOperation := commandName
		if commandName == "plan" {
			expectedOperation = "sync"
		}
		for _, expected := range []string{
			"Project: ",
			"Provider: native",
			"Selection reason: explicit",
			"PHP: ",
			"Composer: ",
			"Operation: " + expectedOperation,
			"Plan digest: sha256:",
		} {
			if !strings.Contains(stdout, expected) {
				t.Fatalf(
					"expected compiled %s output to contain %q, got:\n%s",
					commandName,
					expected,
					stdout,
				)
			}
		}
		if !strings.Contains(stderr, "ELEFANTE_COMPOSER_LOCK_MISSING") {
			t.Fatalf(
				"expected compiled %s warning, got:\n%s",
				commandName,
				stderr,
			)
		}
	}
}

func TestCompiledSyncPreflightFailuresDoNotMutate(t *testing.T) {
	if _, err := exec.LookPath("php"); err != nil {
		t.Skip("local PHP executable is unavailable")
	}
	if _, err := exec.LookPath("composer"); err != nil {
		t.Skip("local Composer executable is unavailable")
	}

	binary := buildBinary(t)
	nativePath := compiledNativeToolPath(t)

	t.Run("approval required", func(t *testing.T) {
		projectRoot := compiledSyncProject(t)
		home := t.TempDir()
		before := readProjectComposer(t, projectRoot)

		exitCode, stdout, stderr := runCompiledWithHome(
			t,
			binary,
			home,
			nativePath,
			"--json",
			"--project", projectRoot,
			"--provider", "native",
			"sync",
		)
		if exitCode != 6 {
			t.Fatalf(
				"expected approval exit 6, got %d\nstdout:\n%s\nstderr:\n%s",
				exitCode,
				stdout,
				stderr,
			)
		}
		assertCompiledErrorCode(t, stdout, model.ErrorApprovalRequired)
		assertCompiledApprovalEvent(t, stdout)
		assertCompiledPreflightUnchanged(t, projectRoot, home, before)
	})

	t.Run("plan mismatch", func(t *testing.T) {
		projectRoot := compiledSyncProject(t)
		home := t.TempDir()
		exitCode, planOutput, stderr := runCompiledWithHome(
			t,
			binary,
			home,
			nativePath,
			"--json",
			"--project", projectRoot,
			"--provider", "native",
			"plan",
		)
		if exitCode != 0 {
			t.Fatalf(
				"build reviewed plan: exit %d\nstdout:\n%s\nstderr:\n%s",
				exitCode,
				planOutput,
				stderr,
			)
		}
		reviewed := planFromEvents(t, decodeCompiledEvents(t, planOutput))
		changed := `{
    "name": "acme/sync-preflight",
    "description": "changed after review",
    "require": {
        "php": ">=8.0",
        "ext-json": "*"
    }
}
`
		if err := os.WriteFile(
			filepath.Join(projectRoot, "composer.json"),
			[]byte(changed),
			0o644,
		); err != nil {
			t.Fatalf("change Composer fixture: %v", err)
		}
		before := readProjectComposer(t, projectRoot)

		exitCode, stdout, stderr := runCompiledWithHome(
			t,
			binary,
			home,
			nativePath,
			"--json",
			"--project", projectRoot,
			"--provider", "native",
			"--approve-plan", reviewed.Digest,
			"sync",
		)
		if exitCode != 7 {
			t.Fatalf(
				"expected mismatch exit 7, got %d\nstdout:\n%s\nstderr:\n%s",
				exitCode,
				stdout,
				stderr,
			)
		}
		assertCompiledErrorCode(t, stdout, model.ErrorPlanMismatch)
		assertCompiledPreflightUnchanged(t, projectRoot, home, before)
	})

	t.Run("offline cache miss", func(t *testing.T) {
		projectRoot := compiledSyncProject(t)
		home := t.TempDir()
		before := readProjectComposer(t, projectRoot)

		exitCode, stdout, stderr := runCompiledWithHome(
			t,
			binary,
			home,
			nativePath,
			"--json",
			"--project", projectRoot,
			"--provider", "native",
			"--offline",
			"--yes",
			"sync",
		)
		if exitCode != 8 {
			t.Fatalf(
				"expected network exit 8, got %d\nstdout:\n%s\nstderr:\n%s",
				exitCode,
				stdout,
				stderr,
			)
		}
		assertCompiledErrorCode(t, stdout, model.ErrorNetwork)
		assertCompiledPreflightUnchanged(t, projectRoot, home, before)
	})
}

func TestCompiledBinaryRegistersDDEVProvider(t *testing.T) {
	binary := buildBinary(t)
	binDirectory := t.TempDir()
	ddevPath := filepath.Join(binDirectory, "ddev")
	ddevScript := `#!/bin/sh
if [ "$1" = "version" ]; then
    printf '%s\n' '{"level":"info","raw":{"DDEV version":"v1.24.8","architecture":"arm64","ddev-environment":"darwin","docker":"29.4.0","docker-platform":"orbstack"}}'
    exit 0
fi
exit 64
`
	if err := os.WriteFile(ddevPath, []byte(ddevScript), 0o755); err != nil {
		t.Fatalf("write fake DDEV executable: %v", err)
	}
	projectRoot := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(projectRoot, "composer.json"),
		[]byte(`{"name":"acme/ddev-proof"}`+"\n"),
		0o644,
	); err != nil {
		t.Fatalf("write DDEV Composer fixture: %v", err)
	}

	command := exec.Command(
		binary,
		"--json",
		"--project",
		projectRoot,
		"--provider",
		"ddev",
		"doctor",
	)
	command.Env = environmentWithPath(os.Environ(), binDirectory)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf(
			"run DDEV doctor: %v\nstdout:\n%s\nstderr:\n%s",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty DDEV doctor stderr, got:\n%s", stderr.String())
	}

	observation := providerObservationFromEvents(
		t,
		decodeCompiledEvents(t, stdout.String()),
		"ddev",
	)
	if !observation.Available ||
		observation.Version != "1.24.8" ||
		observation.State != model.ProviderStateUnconfigured ||
		len(observation.Engines) != 1 ||
		observation.Engines[0].Platform != "orbstack" {
		t.Fatalf("unexpected compiled DDEV observation %#v", observation)
	}
}

func TestCompiledBinaryJSONDoctorCoversEveryFrameworkFixture(t *testing.T) {
	binary := buildBinary(t)
	fixtureRoot := filepath.Join("..", "..", "testdata", "fixtures", "frameworks")
	tests := []struct {
		name     string
		expected []model.FrameworkKind
		conflict bool
	}{
		{
			name:     "laravel-application",
			expected: []model.FrameworkKind{model.FrameworkLaravelApplication},
		},
		{
			name:     "laravel-package",
			expected: []model.FrameworkKind{model.FrameworkLaravelPackage},
		},
		{
			name:     "generic-composer",
			expected: []model.FrameworkKind{model.FrameworkGenericComposer},
		},
		{
			name:     "bedrock-wordpress",
			expected: []model.FrameworkKind{model.FrameworkBedrockWordPress},
		},
		{
			name:     "symfony",
			expected: []model.FrameworkKind{model.FrameworkSymfonyApplication},
		},
		{
			name: "conflicting",
			expected: []model.FrameworkKind{
				model.FrameworkLaravelApplication,
				model.FrameworkSymfonyApplication,
			},
			conflict: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			projectPath, err := filepath.Abs(filepath.Join(fixtureRoot, test.name))
			if err != nil {
				t.Fatalf("resolve fixture path: %v", err)
			}
			facts := decodeDoctorFacts(t, runCompiledDoctor(t, binary, projectPath))

			for _, expected := range test.expected {
				if !containsFramework(facts.Frameworks, expected) {
					t.Fatalf(
						"expected framework %q, got %#v",
						expected,
						facts.Frameworks,
					)
				}
			}
			if test.conflict &&
				!containsDiagnostic(facts.Diagnostics, "ELEFANTE_FRAMEWORK_CONFLICT") {
				t.Fatalf("expected framework conflict, got %#v", facts.Diagnostics)
			}
		})
	}
}

func TestCompiledBinaryJSONDoctorEmitsComposerLockFacts(t *testing.T) {
	binary := buildBinary(t)
	projectRoot := t.TempDir()
	copyComposerFixture(t, projectRoot, "locked-platform", "composer.json")
	copyComposerFixture(t, projectRoot, "locked-platform", "composer.lock")

	first := runCompiledDoctor(t, binary, projectRoot)
	second := runCompiledDoctor(t, binary, projectRoot)
	if first != second {
		t.Fatalf(
			"expected deterministic Composer facts\nfirst:\n%s\nsecond:\n%s",
			first,
			second,
		)
	}

	facts := decodeDoctorFacts(t, first)
	if facts.Composer.Manifest.Name != "acme/locked-platform" {
		t.Errorf(
			"expected Composer package acme/locked-platform, got %q",
			facts.Composer.Manifest.Name,
		)
	}
	if facts.Composer.Lock.Status != model.ComposerLockFresh {
		t.Errorf("expected fresh Composer lock, got %q", facts.Composer.Lock.Status)
	}
	if facts.Composer.Lock.ContentHash == "" ||
		facts.Composer.Lock.ContentHash != facts.Composer.Lock.ExpectedContentHash {
		t.Errorf("expected matching Composer content hashes, got %#v", facts.Composer.Lock)
	}
	if len(facts.Composer.PlatformRequirements) != 6 {
		t.Errorf(
			"expected six root and locked platform requirements, got %#v",
			facts.Composer.PlatformRequirements,
		)
	}
	if len(facts.Composer.PlatformEmulation) != 2 {
		t.Errorf(
			"expected manifest and lock platform emulation facts, got %#v",
			facts.Composer.PlatformEmulation,
		)
	}
	if len(facts.Diagnostics) != 0 {
		t.Errorf("expected no Composer diagnostics, got %#v", facts.Diagnostics)
	}
	if len(facts.InputFingerprints) != 2 {
		t.Fatalf(
			"expected manifest and lock fingerprints, got %#v",
			facts.InputFingerprints,
		)
	}
	if facts.InputFingerprints[0].Kind != "composer_manifest" ||
		facts.InputFingerprints[1].Kind != "composer_lock" {
		t.Errorf("unexpected input fingerprints %#v", facts.InputFingerprints)
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

func environmentWithPath(environment []string, path string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, variable := range environment {
		if strings.HasPrefix(variable, "PATH=") {
			continue
		}
		result = append(result, variable)
	}

	return append(result, "PATH="+path)
}

func runCompiledDoctor(t *testing.T, binary string, projectPath string) string {
	t.Helper()

	command := exec.Command(binary, "--json", "--project", projectPath, "doctor")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) || exitError.ExitCode() != 4 {
			t.Fatalf("run elefante doctor: %v\nstderr:\n%s", err, stderr.String())
		}
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

func runCompiledAnalysis(
	t *testing.T,
	binary string,
	arguments ...string,
) string {
	t.Helper()

	command := exec.Command(binary, arguments...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf(
			"run compiled analysis: %v\nstdout:\n%s\nstderr:\n%s",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected analysis stderr to be empty, got:\n%s", stderr.String())
	}

	return stdout.String()
}

func runCompiledHumanAnalysis(
	t *testing.T,
	binary string,
	arguments ...string,
) (string, string) {
	t.Helper()

	command := exec.Command(binary, arguments...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf(
			"run compiled human analysis: %v\nstdout:\n%s\nstderr:\n%s",
			err,
			stdout.String(),
			stderr.String(),
		)
	}

	return stdout.String(), stderr.String()
}

func decodeDoctorFacts(t *testing.T, content string) model.ProjectFacts {
	t.Helper()

	events := decodeCompiledEvents(t, content)

	return projectFactsFromEvents(t, events)
}

func projectFactsFromEvents(
	t *testing.T,
	events []compiledEvent,
) model.ProjectFacts {
	t.Helper()

	for _, event := range events {
		if event.Type != model.EventFact {
			continue
		}
		var shape map[string]json.RawMessage
		if err := json.Unmarshal(event.Payload, &shape); err != nil {
			t.Fatalf("decode fact shape: %v", err)
		}
		if _, found := shape["identity"]; !found {
			continue
		}
		if _, found := shape["composer"]; !found {
			continue
		}
		var facts model.ProjectFacts
		if err := json.Unmarshal(event.Payload, &facts); err != nil {
			t.Fatalf("decode doctor facts: %v", err)
		}

		return facts
	}
	t.Fatalf("expected project facts event, got %#v", events)

	return model.ProjectFacts{}
}

func providerObservationFromEvents(
	t *testing.T,
	events []compiledEvent,
	name string,
) model.ProviderObservation {
	t.Helper()

	for _, event := range events {
		if event.Type != model.EventFact {
			continue
		}
		var shape map[string]json.RawMessage
		if err := json.Unmarshal(event.Payload, &shape); err != nil {
			t.Fatalf("decode provider fact shape: %v", err)
		}
		if _, found := shape["provider"]; !found {
			continue
		}
		var observation model.ProviderObservation
		if err := json.Unmarshal(event.Payload, &observation); err != nil {
			t.Fatalf("decode provider observation: %v", err)
		}
		if observation.Provider == name &&
			observation.Fingerprint != "" {
			return observation
		}
	}
	t.Fatalf("expected provider observation %q, got %#v", name, events)

	return model.ProviderObservation{}
}

func planFromEvents(t *testing.T, events []compiledEvent) model.Plan {
	t.Helper()

	for _, event := range events {
		if event.Type != model.EventPlan {
			continue
		}
		var builtPlan model.Plan
		if err := json.Unmarshal(event.Payload, &builtPlan); err != nil {
			t.Fatalf("decode plan event: %v", err)
		}

		return builtPlan
	}
	t.Fatalf("expected plan event, got %#v", events)

	return model.Plan{}
}

func containsFramework(
	frameworks []model.FrameworkFact,
	expected model.FrameworkKind,
) bool {
	for _, framework := range frameworks {
		if framework.Kind == expected {
			return true
		}
	}

	return false
}

func containsDiagnostic(
	diagnostics []model.Diagnostic,
	expected string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == expected {
			return true
		}
	}

	return false
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

func compiledSyncProject(t *testing.T) string {
	t.Helper()

	projectRoot := t.TempDir()
	content := `{
    "name": "acme/sync-preflight",
    "require": {
        "php": ">=8.0",
        "ext-json": "*"
    }
}
`
	if err := os.WriteFile(
		filepath.Join(projectRoot, "composer.json"),
		[]byte(content),
		0o644,
	); err != nil {
		t.Fatalf("write synchronization fixture: %v", err)
	}

	return projectRoot
}

func runCompiledWithHome(
	t *testing.T,
	binary string,
	home string,
	nativePath string,
	arguments ...string,
) (int, string, string) {
	t.Helper()

	command := exec.Command(binary, arguments...)
	command.Env = environmentWithOverrides(os.Environ(), map[string]string{
		"HOME":            home,
		"XDG_CONFIG_HOME": filepath.Join(home, "xdg-config"),
		"XDG_CACHE_HOME":  filepath.Join(home, "xdg-cache"),
		"XDG_STATE_HOME":  filepath.Join(home, "xdg-state"),
		"PATH":            nativePath,
	})
	command.Stdin = strings.NewReader("")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if err == nil {
		return 0, stdout.String(), stderr.String()
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("run compiled command: %v", err)
	}

	return exitError.ExitCode(), stdout.String(), stderr.String()
}

func compiledNativeToolPath(t *testing.T) string {
	t.Helper()

	directory := t.TempDir()
	for _, executable := range []string{"php", "composer"} {
		source, err := exec.LookPath(executable)
		if err != nil {
			t.Fatalf("resolve %s executable: %v", executable, err)
		}
		if err := os.Symlink(source, filepath.Join(directory, executable)); err != nil {
			t.Fatalf("link %s executable: %v", executable, err)
		}
	}

	return directory
}

func environmentWithOverrides(
	environment []string,
	overrides map[string]string,
) []string {
	result := make([]string, 0, len(environment)+len(overrides))
	for _, variable := range environment {
		name, _, _ := strings.Cut(variable, "=")
		if _, replaced := overrides[name]; replaced {
			continue
		}
		result = append(result, variable)
	}
	for name, value := range overrides {
		result = append(result, name+"="+value)
	}

	return result
}

func readProjectComposer(t *testing.T, projectRoot string) []byte {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(projectRoot, "composer.json"))
	if err != nil {
		t.Fatalf("read project Composer file: %v", err)
	}

	return content
}

func assertCompiledPreflightUnchanged(
	t *testing.T,
	projectRoot string,
	home string,
	before []byte,
) {
	t.Helper()

	after := readProjectComposer(t, projectRoot)
	if !bytes.Equal(before, after) {
		t.Fatalf("preflight failure changed composer.json")
	}
	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		t.Fatalf("inspect project after preflight: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "composer.json" {
		t.Fatalf("preflight failure changed project contents: %#v", entries)
	}
	for _, stateRoot := range []string{
		filepath.Join(home, "Library", "Application Support", "Elefante"),
		filepath.Join(home, "xdg-config", "elefante"),
		filepath.Join(home, ".elefante"),
	} {
		if _, err := os.Stat(stateRoot); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("preflight failure created Elefante state at %s", stateRoot)
		}
	}
}

func assertCompiledErrorCode(
	t *testing.T,
	content string,
	expected model.ErrorCode,
) {
	t.Helper()

	for _, event := range decodeCompiledEvents(t, content) {
		if event.Type != model.EventError {
			continue
		}
		var commandError model.Error
		if err := json.Unmarshal(event.Payload, &commandError); err != nil {
			t.Fatalf("decode compiled command error: %v", err)
		}
		if commandError.Code != expected {
			t.Fatalf("expected %s, got %#v", expected, commandError)
		}

		return
	}
	t.Fatalf("expected compiled error %s", expected)
}

func assertCompiledApprovalEvent(t *testing.T, content string) {
	t.Helper()

	for _, event := range decodeCompiledEvents(t, content) {
		if event.Type == model.EventApprovalRequired {
			return
		}
	}
	t.Fatal("expected compiled approval_required event")
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

func copyComposerFixture(
	t *testing.T,
	projectRoot string,
	fixture string,
	name string,
) {
	t.Helper()

	source := filepath.Join(
		repositoryRoot(t),
		"testdata",
		"fixtures",
		"composer",
		fixture,
		name,
	)
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read Composer fixture %s: %v", source, err)
	}
	target := filepath.Join(projectRoot, name)
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatalf("write Composer fixture %s: %v", target, err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	return filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
}
