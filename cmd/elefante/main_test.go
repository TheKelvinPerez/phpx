package main_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestCompiledNativeSyncPreservesApprovedComposerSemanticsAndStreams(
	t *testing.T,
) {
	binary := buildBinary(t)
	projectRoot := compiledTrustedSyncProject(t)
	home := t.TempDir()
	invocationLog := filepath.Join(t.TempDir(), "composer-invocations.log")
	nativePath := compiledNativeSyncToolPath(t)
	t.Setenv("ELEFANTE_TEST_COMPOSER_LOG", invocationLog)
	manifestPath := filepath.Join(projectRoot, "composer.json")
	lockPath := filepath.Join(projectRoot, "composer.lock")
	manifestBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read frozen manifest before sync: %v", err)
	}
	lockBefore, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read frozen lock before sync: %v", err)
	}

	exitCode, stdout, stderr := runCompiledWithHome(
		t,
		binary,
		home,
		nativePath,
		"--json",
		"--project", projectRoot,
		"--provider", "native",
		"--frozen",
		"--non-interactive",
		"sync",
	)
	if exitCode != 6 {
		t.Fatalf(
			"expected trusted sync approval exit 6, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	if _, err := os.Stat(invocationLog); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unapproved Composer code reached execution: %v", err)
	}
	var approval model.ApprovalRequiredPayload
	for _, event := range decodeCompiledEvents(t, stdout) {
		if event.Type != model.EventApprovalRequired {
			continue
		}
		if err := json.Unmarshal(event.Payload, &approval); err != nil {
			t.Fatalf("decode trusted sync approval: %v", err)
		}
	}
	trustClasses := make(map[model.TrustClass]bool)
	for _, requirement := range approval.Trust {
		trustClasses[requirement.Class] = true
	}
	if !trustClasses[model.TrustComposerScripts] ||
		!trustClasses[model.TrustComposerPlugins] {
		t.Fatalf(
			"trusted sync approval omitted Composer code classes %#v",
			approval,
		)
	}

	exitCode, stdout, stderr = runCompiledWithHome(
		t,
		binary,
		home,
		nativePath,
		"--project", projectRoot,
		"--provider", "native",
		"--frozen",
		"--non-interactive",
		"--yes",
		"sync",
	)
	if exitCode != 0 {
		t.Fatalf(
			"expected approved native sync exit zero, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	for _, expected := range []string{
		"composer-install-stdout",
		"composer-platform-stdout",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf(
				"expected native sync stdout to stream %q, got:\n%s",
				expected,
				stdout,
			)
		}
	}
	for _, expected := range []string{
		"composer-install-stderr",
		"composer-platform-stderr",
	} {
		if !strings.Contains(stderr, expected) {
			t.Fatalf(
				"expected native sync stderr to stream %q, got:\n%s",
				expected,
				stderr,
			)
		}
	}

	invocations, err := os.ReadFile(invocationLog)
	if err != nil {
		t.Fatalf("read Composer invocation log: %v", err)
	}
	expectedInvocations := "install --no-interaction\ncheck-platform-reqs\n"
	if string(invocations) != expectedInvocations {
		t.Fatalf(
			"approved frozen sync changed Composer semantics\nexpected:\n%s\ngot:\n%s",
			expectedInvocations,
			invocations,
		)
	}
	if strings.Contains(string(invocations), "--no-scripts") ||
		strings.Contains(string(invocations), "--no-plugins") {
		t.Fatalf(
			"approved Composer scripts or plugins were disabled:\n%s",
			invocations,
		)
	}
	manifestAfter, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read frozen manifest after sync: %v", err)
	}
	lockAfter, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read frozen lock after sync: %v", err)
	}
	if !bytes.Equal(manifestAfter, manifestBefore) ||
		!bytes.Equal(lockAfter, lockBefore) {
		t.Fatal("frozen native sync changed Composer project files")
	}
}

func TestCompiledNativeSyncCompletesLockedSupportedFixture(t *testing.T) {
	if _, err := exec.LookPath("php"); err != nil {
		t.Skip("local PHP executable is unavailable")
	}
	if _, err := exec.LookPath("composer"); err != nil {
		t.Skip("local Composer executable is unavailable")
	}

	binary := buildBinary(t)
	projectRoot := t.TempDir()
	for _, name := range []string{"composer.json", "composer.lock"} {
		copyRuntimeFixture(t, projectRoot, "native-sync", name)
	}
	home := t.TempDir()
	nativePath := compiledNativeToolPath(t)
	lockPath := filepath.Join(projectRoot, "composer.lock")
	lockBefore, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read supported fixture lock before sync: %v", err)
	}

	exitCode, stdout, stderr := runCompiledWithHome(
		t,
		binary,
		home,
		nativePath,
		"--project", projectRoot,
		"--provider", "native",
		"--frozen",
		"--non-interactive",
		"--yes",
		"sync",
	)
	if exitCode != 0 {
		t.Fatalf(
			"expected locked native sync exit zero, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	combinedStreams := stdout + stderr
	if !strings.Contains(
		combinedStreams,
		"Installing dependencies from lock file",
	) ||
		!strings.Contains(
			combinedStreams,
			"checking platform requirements",
		) {
		t.Fatalf(
			"native sync did not expose install and platform verification\nstdout:\n%s\nstderr:\n%s",
			stdout,
			stderr,
		)
	}
	if _, err := os.Stat(
		filepath.Join(projectRoot, "vendor", "autoload.php"),
	); err != nil {
		t.Fatalf("native sync did not install the locked fixture: %v", err)
	}
	lockAfter, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read supported fixture lock after sync: %v", err)
	}
	if !bytes.Equal(lockAfter, lockBefore) {
		t.Fatal("frozen supported fixture sync changed composer.lock")
	}
}

func TestCompiledNativeRunPreservesArgumentsDirectoryAndRawStreams(
	t *testing.T,
) {
	binary := buildBinary(t)
	projectRoot := compiledRunProject(t)
	home := t.TempDir()
	nativePath := compiledNativeRunToolPath(t)

	exitCode, stdout, stderr := runCompiledWithHome(
		t,
		binary,
		home,
		nativePath,
		"--project", projectRoot,
		"--provider", "native",
		"run",
		"--",
		"child-proof",
		"space value",
		"$(printf unsafe)",
		"",
	)
	if exitCode != 0 {
		t.Fatalf(
			"expected native run exit zero, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	resolvedProjectRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		t.Fatalf("resolve run fixture root: %v", err)
	}
	expectedStdout := "child-arguments=<space value>|<$(printf unsafe)>|<>\n" +
		"child-directory=" + resolvedProjectRoot + "\n"
	if stdout != expectedStdout {
		t.Fatalf(
			"native run changed raw stdout\nexpected:\n%s\ngot:\n%s",
			expectedStdout,
			stdout,
		)
	}
	if stderr != "child-stderr\n" {
		t.Fatalf("native run changed raw stderr: %q", stderr)
	}
}

func TestCompiledNativeRunRequiresSeparatorAndPreservesChildExit(t *testing.T) {
	binary := buildBinary(t)
	projectRoot := compiledRunProject(t)
	home := t.TempDir()
	nativePath := compiledNativeRunToolPath(t)

	exitCode, stdout, stderr := runCompiledWithHome(
		t,
		binary,
		home,
		nativePath,
		"--project", projectRoot,
		"--provider", "native",
		"run",
		"child-proof",
	)
	if exitCode != 2 {
		t.Fatalf(
			"expected missing separator exit 2, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	if stdout != "" ||
		!strings.Contains(stderr, "requires the command separator") {
		t.Fatalf(
			"unexpected missing separator response\nstdout:\n%s\nstderr:\n%s",
			stdout,
			stderr,
		)
	}

	exitCode, stdout, stderr = runCompiledWithHome(
		t,
		binary,
		home,
		nativePath,
		"--project", projectRoot,
		"--provider", "native",
		"run",
		"--",
		"child-exit",
	)
	if exitCode != 37 {
		t.Fatalf(
			"expected child exit 37, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	if stdout != "child-before-exit\n" ||
		stderr != "child-error-before-exit\n" {
		t.Fatalf(
			"nonzero child streams changed\nstdout:\n%s\nstderr:\n%s",
			stdout,
			stderr,
		)
	}

	exitCode, stdout, stderr = runCompiledWithHome(
		t,
		binary,
		home,
		nativePath,
		"--json",
		"--project", projectRoot,
		"--provider", "native",
		"run",
		"--",
		"child-exit",
	)
	if exitCode != 37 || stderr != "" {
		t.Fatalf(
			"expected machine child exit 37, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	events := decodeCompiledEvents(t, stdout)
	var completed model.CompletedPayload
	if err := json.Unmarshal(
		events[len(events)-1].Payload,
		&completed,
	); err != nil {
		t.Fatalf("decode nonzero child completion: %v", err)
	}
	if completed.Exit.Origin != model.ExitOriginChild ||
		completed.Exit.Code != 37 {
		t.Fatalf("machine nonzero exit lost child origin %#v", completed)
	}
	if got := reconstructChildStream(
		t,
		events,
		model.EventStdout,
	); string(got) != "child-before-exit\n" {
		t.Fatalf("machine nonzero stdout changed: %q", got)
	}
	if got := reconstructChildStream(
		t,
		events,
		model.EventStderr,
	); string(got) != "child-error-before-exit\n" {
		t.Fatalf("machine nonzero stderr changed: %q", got)
	}
}

func TestCompiledNativeJSONRunEncodesReconstructableChildStreams(t *testing.T) {
	binary := buildBinary(t)
	projectRoot := compiledRunProject(t)
	home := t.TempDir()
	nativePath := compiledNativeRunToolPath(t)

	exitCode, stdout, stderr := runCompiledWithHome(
		t,
		binary,
		home,
		nativePath,
		"--json",
		"--project", projectRoot,
		"--provider", "native",
		"run",
		"--",
		"child-binary",
	)
	if exitCode != 0 {
		t.Fatalf(
			"expected JSON child exit zero, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	if stderr != "" {
		t.Fatalf("machine mode leaked raw standard error: %q", stderr)
	}

	events := decodeCompiledEvents(t, stdout)
	if len(events) < 4 ||
		events[0].Type != model.EventStarted ||
		events[len(events)-1].Type != model.EventCompleted {
		t.Fatalf("unexpected JSON run event sequence %#v", events)
	}
	for index, event := range events {
		if event.Sequence != uint64(index+1) ||
			event.Command != "run" {
			t.Fatalf("unexpected JSON run event %d: %#v", index+1, event)
		}
	}
	reconstructedStdout := reconstructChildStream(
		t,
		events,
		model.EventStdout,
	)
	reconstructedStderr := reconstructChildStream(
		t,
		events,
		model.EventStderr,
	)
	expectedStdout := append(
		[]byte("utf8-line\n"),
		[]byte{0xff, 0x00, 'b', 'i', 'n', 'a', 'r', 'y', '-', 't', 'a', 'i', 'l', '\n'}...,
	)
	if !bytes.Equal(reconstructedStdout, expectedStdout) {
		t.Fatalf(
			"machine stdout events cannot reconstruct child bytes\nexpected: %v\ngot:      %v",
			expectedStdout,
			reconstructedStdout,
		)
	}
	expectedStderr := []byte("child-binary-stderr\n")
	if !bytes.Equal(reconstructedStderr, expectedStderr) {
		t.Fatalf(
			"machine stderr events cannot reconstruct child bytes\nexpected: %v\ngot:      %v",
			expectedStderr,
			reconstructedStderr,
		)
	}
	var completed model.CompletedPayload
	if err := json.Unmarshal(
		events[len(events)-1].Payload,
		&completed,
	); err != nil {
		t.Fatalf("decode JSON run completion: %v", err)
	}
	if completed.Exit.Origin != model.ExitOriginChild ||
		completed.Exit.Code != 0 {
		t.Fatalf("JSON run did not identify the child exit %#v", completed)
	}
}

func TestCompiledNativeRunForwardsInterruptToChild(t *testing.T) {
	binary := buildBinary(t)
	projectRoot := compiledRunProject(t)
	home := t.TempDir()
	nativePath := compiledNativeRunToolPath(t)
	readyPath := filepath.Join(t.TempDir(), "child-ready")

	command := exec.Command(
		binary,
		"--project", projectRoot,
		"--provider", "native",
		"run",
		"--",
		"child-signal",
	)
	command.Env = environmentWithOverrides(os.Environ(), map[string]string{
		"ELEFANTE_TEST_READY_FILE": readyPath,
		"HOME":                     home,
		"PATH":                     nativePath,
		"XDG_CACHE_HOME":           filepath.Join(home, "xdg-cache"),
		"XDG_CONFIG_HOME":          filepath.Join(home, "xdg-config"),
		"XDG_STATE_HOME":           filepath.Join(home, "xdg-state"),
	})
	command.Stdin = strings.NewReader("")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Start(); err != nil {
		t.Fatalf("start signal forwarding proof: %v", err)
	}
	t.Cleanup(func() {
		_ = command.Process.Kill()
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(readyPath); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("inspect child readiness: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatalf(
				"child did not become ready\nstdout:\n%s\nstderr:\n%s",
				stdout.String(),
				stderr.String(),
			)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := command.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("interrupt Elefante process: %v", err)
	}
	err := command.Wait()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf(
			"expected forwarded child exit, got %v\nstdout:\n%s\nstderr:\n%s",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	if exitError.ExitCode() != 23 {
		t.Fatalf(
			"expected interrupt handled child exit 23, got %d\nstdout:\n%s\nstderr:\n%s",
			exitError.ExitCode(),
			stdout.String(),
			stderr.String(),
		)
	}
	if stdout.String() != "child-ready\nchild-signal=interrupt\n" ||
		stderr.Len() != 0 {
		t.Fatalf(
			"forwarded interrupt changed child streams\nstdout:\n%s\nstderr:\n%s",
			stdout.String(),
			stderr.String(),
		)
	}
}

func TestCompiledNativeRunPreservesSignalExit(t *testing.T) {
	binary := buildBinary(t)
	projectRoot := compiledRunProject(t)
	home := t.TempDir()
	nativePath := compiledNativeRunToolPath(t)

	exitCode, stdout, stderr := runCompiledWithHome(
		t,
		binary,
		home,
		nativePath,
		"--project", projectRoot,
		"--provider", "native",
		"run",
		"--",
		"child-terminate",
	)
	if exitCode != 143 {
		t.Fatalf(
			"expected signal exit 143, got %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	if stdout != "child-before-signal\n" || stderr != "" {
		t.Fatalf(
			"signaled child streams changed\nstdout:\n%s\nstderr:\n%s",
			stdout,
			stderr,
		)
	}
}

func TestCompiledNativeJSONRunReconstructsLargeOutput(t *testing.T) {
	binary := buildBinary(t)
	projectRoot := compiledRunProject(t)
	home := t.TempDir()
	nativePath := compiledNativeRunToolPath(t)

	exitCode, stdout, stderr := runCompiledWithHome(
		t,
		binary,
		home,
		nativePath,
		"--json",
		"--project", projectRoot,
		"--provider", "native",
		"run",
		"--",
		"child-large",
	)
	if exitCode != 0 || stderr != "" {
		t.Fatalf(
			"large JSON child failed with exit %d\nstdout:\n%s\nstderr:\n%s",
			exitCode,
			stdout,
			stderr,
		)
	}
	reconstructed := reconstructChildStream(
		t,
		decodeCompiledEvents(t, stdout),
		model.EventStdout,
	)
	line := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ\n"
	expected := []byte(strings.Repeat(line, 4096))
	if !bytes.Equal(reconstructed, expected) {
		t.Fatalf(
			"large child output changed, expected %d bytes, got %d",
			len(expected),
			len(reconstructed),
		)
	}
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

func compiledTrustedSyncProject(t *testing.T) string {
	t.Helper()

	projectRoot := t.TempDir()
	manifest := `{
    "name": "acme/trusted-sync",
    "require": {
        "php": ">=8.0",
        "ext-json": "*"
    },
    "scripts": {
        "post-install-cmd": "Acme\\Setup::run"
    }
}
`
	lock := `{
    "content-hash": "00000000000000000000000000000000",
    "packages": [
        {
            "name": "acme/plugin",
            "version": "1.0.0",
            "type": "composer-plugin",
            "require": {
                "php": ">=8.0"
            }
        }
    ],
    "packages-dev": [],
    "platform": {},
    "platform-dev": {},
    "plugin-api-version": "2.6.0"
}
`
	for name, content := range map[string]string{
		"composer.json": manifest,
		"composer.lock": lock,
	} {
		if err := os.WriteFile(
			filepath.Join(projectRoot, name),
			[]byte(content),
			0o644,
		); err != nil {
			t.Fatalf("write trusted synchronization fixture %s: %v", name, err)
		}
	}

	return projectRoot
}

func compiledRunProject(t *testing.T) string {
	t.Helper()

	projectRoot := t.TempDir()
	content := `{
    "name": "acme/run-proof",
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
		t.Fatalf("write run fixture: %v", err)
	}

	return projectRoot
}

func compiledNativeSyncToolPath(t *testing.T) string {
	t.Helper()

	directory := t.TempDir()
	php := `#!/bin/sh
if [ "$1" = "-r" ]; then
    printf '%s\n' '{"version":"8.5.0","sapi":"cli","binary":"php","extensions":[{"name":"json","version":"8.5.0"}]}'
    exit 0
fi
exit 64
`
	composer := `#!/bin/sh
if [ "$1" = "--version" ]; then
    printf '%s\n' 'Composer version 2.9.5 2026-01-29 11:40:53'
    exit 0
fi
printf '%s\n' "$*" >> "$ELEFANTE_TEST_COMPOSER_LOG"
if [ "$1" = "install" ]; then
    printf '%s\n' 'composer-install-stdout'
    printf '%s\n' 'composer-install-stderr' >&2
    exit 0
fi
if [ "$1" = "check-platform-reqs" ]; then
    printf '%s\n' 'composer-platform-stdout'
    printf '%s\n' 'composer-platform-stderr' >&2
    exit 0
fi
exit 64
`
	for name, content := range map[string]string{
		"php":      php,
		"composer": composer,
	} {
		if err := os.WriteFile(
			filepath.Join(directory, name),
			[]byte(content),
			0o755,
		); err != nil {
			t.Fatalf("write fake native %s executable: %v", name, err)
		}
	}

	return directory
}

func compiledNativeRunToolPath(t *testing.T) string {
	t.Helper()

	directory := t.TempDir()
	php := `#!/bin/sh
if [ "$1" = "-r" ]; then
    printf '%s\n' '{"version":"8.5.0","sapi":"cli","binary":"php","extensions":[{"name":"json","version":"8.5.0"}]}'
    exit 0
fi
exit 64
`
	composer := `#!/bin/sh
if [ "$1" = "--version" ]; then
    printf '%s\n' 'Composer version 2.9.5 2026-01-29 11:40:53'
    exit 0
fi
exit 64
`
	child := `#!/bin/sh
printf 'child-arguments=<%s>|<%s>|<%s>\n' "$1" "$2" "$3"
printf 'child-directory=%s\n' "$(pwd -P)"
printf '%s\n' 'child-stderr' >&2
`
	childExit := `#!/bin/sh
printf '%s\n' 'child-before-exit'
printf '%s\n' 'child-error-before-exit' >&2
exit 37
`
	childBinary := `#!/bin/sh
printf '%s\n' 'utf8-line'
printf '\377\000binary-tail\n'
printf '%s\n' 'child-binary-stderr' >&2
`
	childSignal := `#!/bin/sh
trap 'printf "%s\n" "child-signal=interrupt"; exit 23' INT
trap 'printf "%s\n" "child-signal=terminated"; exit 24' TERM
printf '%s\n' 'child-ready'
: > "$ELEFANTE_TEST_READY_FILE"
while :; do
    /bin/sleep 1
done
`
	childTerminate := `#!/bin/sh
printf '%s\n' 'child-before-signal'
kill -TERM $$
`
	childLarge := `#!/bin/sh
index=0
while [ "$index" -lt 4096 ]; do
    printf '%s\n' '0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ'
    index=$((index + 1))
done
`
	for name, content := range map[string]string{
		"php":             php,
		"composer":        composer,
		"child-proof":     child,
		"child-exit":      childExit,
		"child-binary":    childBinary,
		"child-signal":    childSignal,
		"child-terminate": childTerminate,
		"child-large":     childLarge,
	} {
		if err := os.WriteFile(
			filepath.Join(directory, name),
			[]byte(content),
			0o755,
		); err != nil {
			t.Fatalf("write fake native %s executable: %v", name, err)
		}
	}

	return directory
}

func reconstructChildStream(
	t *testing.T,
	events []compiledEvent,
	eventType model.EventType,
) []byte {
	t.Helper()

	var reconstructed []byte
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		var payload struct {
			Encoding string `json:"encoding"`
			Data     string `json:"data"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode %s payload: %v", eventType, err)
		}
		switch payload.Encoding {
		case "utf8":
			reconstructed = append(reconstructed, []byte(payload.Data)...)
		case "base64":
			decoded, err := base64.StdEncoding.DecodeString(payload.Data)
			if err != nil {
				t.Fatalf("decode %s base64 payload: %v", eventType, err)
			}
			reconstructed = append(reconstructed, decoded...)
		default:
			t.Fatalf(
				"unexpected %s payload encoding %q",
				eventType,
				payload.Encoding,
			)
		}
	}

	return reconstructed
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

func copyRuntimeFixture(
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
		"runtime",
		fixture,
		name,
	)
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read runtime fixture %s: %v", source, err)
	}
	if err := os.WriteFile(
		filepath.Join(projectRoot, name),
		content,
		0o644,
	); err != nil {
		t.Fatalf("write runtime fixture %s: %v", name, err)
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
