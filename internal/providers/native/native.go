package native

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/elefantephp/elefante/internal/executor"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
)

const providerVersion = "native/v1"

const phpInspectionScript = `$extensions = [];
foreach (get_loaded_extensions() as $name) {
    $version = phpversion($name);
    $extensions[] = [
        'name' => $name,
        'version' => $version === false ? '' : $version,
    ];
}
echo json_encode([
    'version' => PHP_VERSION,
    'sapi' => PHP_SAPI,
    'binary' => PHP_BINARY,
    'extensions' => $extensions,
], JSON_THROW_ON_ERROR);`

type Dependencies struct {
	Runner       executor.Runner
	GOOS         string
	GOARCH       string
	ProviderPath string
}

type Provider struct {
	runner       executor.Runner
	goos         string
	goarch       string
	providerPath string
}

func New(dependencies ...Dependencies) *Provider {
	var configured Dependencies
	if len(dependencies) > 0 {
		configured = dependencies[0]
	}
	if configured.Runner == nil {
		configured.Runner = executor.OSRunner{}
	}
	if configured.GOOS == "" {
		configured.GOOS = runtime.GOOS
	}
	if configured.GOARCH == "" {
		configured.GOARCH = runtime.GOARCH
	}

	return &Provider{
		runner:       configured.Runner,
		goos:         configured.GOOS,
		goarch:       configured.GOARCH,
		providerPath: configured.ProviderPath,
	}
}

func (provider *Provider) Name() string {
	return "native"
}

func (provider *Provider) Inspect(
	ctx context.Context,
	_ providers.InspectRequest,
) (model.ProviderObservation, error) {
	observation := model.ProviderObservation{
		Provider:     provider.Name(),
		Available:    true,
		Version:      providerVersion,
		Platform:     provider.goos,
		Architecture: provider.goarch,
		Capabilities: []model.Capability{
			model.CapabilityExecuteCommand,
			model.CapabilityInspectComposer,
			model.CapabilityInspectExtensions,
			model.CapabilityInspectPlatform,
			model.CapabilityInspectRuntime,
		},
		Runtimes:    []model.RuntimeObservation{},
		Composer:    []model.ComposerObservation{},
		Extensions:  []model.ExtensionObservation{},
		Diagnostics: []model.Diagnostic{},
	}

	phpPath, err := provider.runner.LookPath("php")
	switch {
	case err == nil:
		phpPath = canonicalExecutablePath(phpPath)
		runtimeObservation, extensions, err := provider.inspectPHP(ctx, phpPath)
		if err != nil {
			return model.ProviderObservation{}, err
		}
		observation.Runtimes = append(observation.Runtimes, runtimeObservation)
		observation.Extensions = append(observation.Extensions, extensions...)
	case executableNotFound(err):
		observation.Diagnostics = append(
			observation.Diagnostics,
			missingExecutableDiagnostic(
				"ELEFANTE_NATIVE_PHP_NOT_FOUND",
				"PHP was not found in the native process environment.",
				"php",
			),
		)
	default:
		return model.ProviderObservation{}, providerError(
			"Could not inspect the native PHP executable.",
			err,
		)
	}

	composerPath, err := provider.runner.LookPath("composer")
	switch {
	case err == nil:
		composerPath = canonicalExecutablePath(composerPath)
		composer, err := provider.inspectComposer(ctx, composerPath)
		if err != nil {
			return model.ProviderObservation{}, err
		}
		observation.Composer = append(observation.Composer, composer)
	case executableNotFound(err):
		observation.Diagnostics = append(
			observation.Diagnostics,
			missingExecutableDiagnostic(
				"ELEFANTE_NATIVE_COMPOSER_NOT_FOUND",
				"Composer was not found in the native process environment.",
				"composer",
			),
		)
	default:
		return model.ProviderObservation{}, providerError(
			"Could not inspect the native Composer executable.",
			err,
		)
	}

	sortDiagnostics(observation.Diagnostics)
	observation.Fingerprint = observationFingerprint(
		observation,
		provider.providerPath,
	)

	return observation, nil
}

type phpInspection struct {
	Version    string `json:"version"`
	SAPI       string `json:"sapi"`
	Binary     string `json:"binary"`
	Extensions []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"extensions"`
}

func (provider *Provider) inspectPHP(
	ctx context.Context,
	path string,
) (
	model.RuntimeObservation,
	[]model.ExtensionObservation,
	error,
) {
	output, err := provider.runner.Output(ctx, executor.Command{
		Executable: path,
		Arguments:  []string{"-r", phpInspectionScript},
	})
	if err != nil {
		return model.RuntimeObservation{}, nil, providerError(
			"Could not query the native PHP runtime.",
			err,
		)
	}

	var inspected phpInspection
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&inspected); err != nil {
		return model.RuntimeObservation{}, nil, providerError(
			"Native PHP returned an invalid inspection response.",
			err,
		)
	}
	var trailing struct{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return model.RuntimeObservation{}, nil, providerError(
			"Native PHP returned an invalid inspection response.",
			errors.New("inspection response contained trailing data"),
		)
	}
	if strings.TrimSpace(inspected.Version) == "" ||
		strings.TrimSpace(inspected.SAPI) == "" {
		return model.RuntimeObservation{}, nil, providerError(
			"Native PHP returned an incomplete inspection response.",
			errors.New("PHP version and SAPI are required"),
		)
	}

	source := executableSource(path)
	extensions := make(
		[]model.ExtensionObservation,
		0,
		len(inspected.Extensions),
	)
	for _, inspectedExtension := range inspected.Extensions {
		name := normalizeExtensionName(inspectedExtension.Name)
		if name == "" {
			continue
		}
		extensions = append(extensions, model.ExtensionObservation{
			Name:      name,
			Version:   inspectedExtension.Version,
			Available: true,
			Source:    source,
		})
	}
	sort.Slice(extensions, func(left int, right int) bool {
		return extensions[left].Name < extensions[right].Name
	})

	return model.RuntimeObservation{
		Name:    "php",
		Version: inspected.Version,
		SAPI:    inspected.SAPI,
		Source:  source,
	}, extensions, nil
}

func normalizeExtensionName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.Join(strings.Fields(name), "-")
	if name == "" || strings.HasPrefix(name, "ext-") {
		return name
	}

	return "ext-" + name
}

func (provider *Provider) Plan(
	_ context.Context,
	_ providers.ProviderPlanRequest,
) (providers.ProviderPlan, error) {
	return providers.ProviderPlan{
		Actions:     []model.PlanAction{},
		Diagnostics: []model.Diagnostic{},
	}, nil
}

func (provider *Provider) Apply(
	_ context.Context,
	action providers.ProviderAction,
	_ providers.ActionRuntime,
) (providers.ActionResult, error) {
	commandError := model.NewError(
		model.ErrorProvider,
		"The native provider does not apply environment mutations.",
	)
	commandError.Provider = provider.Name()
	commandError.Details = []model.ErrorDetail{
		{Name: "action", Value: string(action.Action.Kind)},
	}

	return providers.ActionResult{}, commandError
}

func (provider *Provider) ExecutionSpec(
	_ context.Context,
	request providers.ExecutionRequest,
) (providers.ExecutionSpec, error) {
	if strings.TrimSpace(request.Executable) == "" {
		return providers.ExecutionSpec{}, model.NewError(
			model.ErrorUsage,
			"Native execution requires an executable.",
		)
	}
	path, err := provider.runner.LookPath(request.Executable)
	if err != nil {
		return providers.ExecutionSpec{}, providerError(
			fmt.Sprintf(
				"Could not find native executable %q.",
				request.Executable,
			),
			err,
		)
	}

	return providers.ExecutionSpec{
		Executable:       canonicalExecutablePath(path),
		Arguments:        append([]string(nil), request.Arguments...),
		WorkingDirectory: request.WorkingDirectory,
		Environment:      append([]string(nil), request.Environment...),
		InputMode:        providers.InputInherit,
		OutputMode:       providers.OutputStream,
	}, nil
}

func (provider *Provider) inspectComposer(
	ctx context.Context,
	path string,
) (model.ComposerObservation, error) {
	output, err := provider.runner.Output(ctx, executor.Command{
		Executable: path,
		Arguments:  []string{"--version", "--no-ansi"},
		Environment: []string{
			"COMPOSER_NO_INTERACTION=1",
		},
	})
	if err != nil {
		return model.ComposerObservation{}, providerError(
			"Could not query the native Composer version.",
			err,
		)
	}
	version, err := parseComposerVersion(output)
	if err != nil {
		return model.ComposerObservation{}, providerError(
			"Native Composer returned an invalid version response.",
			err,
		)
	}
	source := executableSource(path)

	return model.ComposerObservation{
		Version:   version,
		Source:    "system",
		Path:      path,
		Identity:  executableIdentity(path, version),
		Reference: source,
	}, nil
}

var composerVersionPattern = regexp.MustCompile(
	`(?m)^Composer version ([^[:space:]]+)`,
)

func parseComposerVersion(output []byte) (string, error) {
	match := composerVersionPattern.FindSubmatch(output)
	if len(match) != 2 {
		return "", errors.New("Composer version line was not found")
	}

	return string(match[1]), nil
}

func executableNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound)
}

func canonicalExecutablePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err == nil {
		path = absolute
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	}

	return filepath.Clean(path)
}

func executableSource(path string) model.SourceReference {
	return model.SourceReference{
		Path: path,
		Kind: "provider_executable",
	}
}

func executableIdentity(path string, version string) string {
	sum := sha256.Sum256([]byte(path + "\x00" + version))

	return "sha256:" + hex.EncodeToString(sum[:])
}

func missingExecutableDiagnostic(
	code string,
	message string,
	executable string,
) model.Diagnostic {
	return model.Diagnostic{
		Code:     code,
		Severity: model.SeverityError,
		Message:  message,
		Hint:     fmt.Sprintf("Add %s to PATH or select another provider.", executable),
		Sources: []model.SourceReference{
			{
				Path:  "PATH",
				Kind:  "process_environment",
				Field: executable,
			},
		},
		Provider: "native",
	}
}

func providerError(message string, cause error) *model.Error {
	commandError := model.WrapError(
		model.ErrorProvider,
		message,
		cause,
	)
	commandError.Provider = "native"
	commandError.Retryable = true

	return commandError
}

func observationFingerprint(
	observation model.ProviderObservation,
	providerPath string,
) string {
	type stableDiagnostic struct {
		Code      string                  `json:"code"`
		Severity  model.Severity          `json:"severity"`
		Sources   []model.SourceReference `json:"sources,omitempty"`
		Provider  string                  `json:"provider,omitempty"`
		Retryable bool                    `json:"retryable"`
	}
	canonical := struct {
		Schema       string                       `json:"schema"`
		ProviderPath string                       `json:"provider_path,omitempty"`
		Provider     string                       `json:"provider"`
		Version      string                       `json:"version"`
		Platform     string                       `json:"platform"`
		Architecture string                       `json:"architecture"`
		Capabilities []model.Capability           `json:"capabilities"`
		Runtimes     []model.RuntimeObservation   `json:"runtimes"`
		Composer     []model.ComposerObservation  `json:"composer"`
		Extensions   []model.ExtensionObservation `json:"extensions"`
		Diagnostics  []stableDiagnostic           `json:"diagnostics"`
	}{
		Schema:       "elefante.provider-observation/v1",
		ProviderPath: providerPath,
		Provider:     observation.Provider,
		Version:      observation.Version,
		Platform:     observation.Platform,
		Architecture: observation.Architecture,
		Capabilities: append(
			[]model.Capability(nil),
			observation.Capabilities...,
		),
		Runtimes: append(
			[]model.RuntimeObservation(nil),
			observation.Runtimes...,
		),
		Composer: append(
			[]model.ComposerObservation(nil),
			observation.Composer...,
		),
		Extensions: append(
			[]model.ExtensionObservation(nil),
			observation.Extensions...,
		),
	}
	for _, diagnostic := range observation.Diagnostics {
		canonical.Diagnostics = append(
			canonical.Diagnostics,
			stableDiagnostic{
				Code:      diagnostic.Code,
				Severity:  diagnostic.Severity,
				Sources:   diagnostic.Sources,
				Provider:  diagnostic.Provider,
				Retryable: diagnostic.Retryable,
			},
		)
	}
	encoded, _ := json.Marshal(canonical)
	sum := sha256.Sum256(encoded)

	return "sha256:" + hex.EncodeToString(sum[:])
}

func sortDiagnostics(diagnostics []model.Diagnostic) {
	sort.Slice(diagnostics, func(left int, right int) bool {
		return diagnostics[left].Code < diagnostics[right].Code
	})
}
