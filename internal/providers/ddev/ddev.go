package ddev

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/elefantephp/elefante/internal/executor"
	"github.com/elefantephp/elefante/internal/model"
	"github.com/elefantephp/elefante/internal/providers"
)

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

type ConfigExists func(string) (bool, error)

type Dependencies struct {
	Runner       executor.Runner
	ConfigExists ConfigExists
}

type Provider struct {
	runner       executor.Runner
	configExists ConfigExists
}

func New(dependencies ...Dependencies) *Provider {
	var configured Dependencies
	if len(dependencies) > 0 {
		configured = dependencies[0]
	}
	if configured.Runner == nil {
		configured.Runner = executor.OSRunner{}
	}
	if configured.ConfigExists == nil {
		configured.ConfigExists = pathExists
	}

	return &Provider{
		runner:       configured.Runner,
		configExists: configured.ConfigExists,
	}
}

func (provider *Provider) Name() string {
	return "ddev"
}

func (provider *Provider) Inspect(
	ctx context.Context,
	request providers.InspectRequest,
) (model.ProviderObservation, error) {
	observation := emptyObservation()
	path, err := provider.runner.LookPath("ddev")
	switch {
	case err == nil:
		path = canonicalExecutablePath(path)
		observation.Available = true
		observation.Capabilities = []model.Capability{
			model.CapabilityExecuteCommand,
			model.CapabilityInspectComposer,
			model.CapabilityInspectExtensions,
			model.CapabilityInspectPlatform,
			model.CapabilityInspectRuntime,
			model.CapabilityInstallExtension,
			model.CapabilityInstallRuntime,
			model.CapabilityStartProvider,
		}
		version, engine, err := provider.inspectVersion(ctx, path)
		if err != nil {
			return model.ProviderObservation{}, err
		}
		observation.Version = version
		observation.Platform = engine.environment
		observation.Architecture = engine.architecture
		observation.Engines = append(
			observation.Engines,
			model.EngineObservation{
				Name:     "docker",
				Version:  engine.version,
				Platform: engine.platform,
				Source: model.SourceReference{
					Path: path,
					Kind: "provider_executable",
				},
			},
		)

		configPath := filepath.Join(
			request.Project.ComposerRoot,
			".ddev",
			"config.yaml",
		)
		exists, err := provider.configExists(configPath)
		if err != nil {
			return model.ProviderObservation{}, providerError(
				"Could not inspect the DDEV project configuration.",
				err,
			)
		}
		if !exists {
			observation.State = model.ProviderStateUnconfigured
			observation.Diagnostics = append(
				observation.Diagnostics,
				model.Diagnostic{
					Code:     "ELEFANTE_DDEV_CONFIG_MISSING",
					Severity: model.SeverityWarning,
					Message:  "The project has no DDEV configuration.",
					Hint:     "Review the DDEV configuration action in elefante plan.",
					Sources: []model.SourceReference{
						{
							Path: configPath,
							Kind: "provider_config",
						},
					},
					Provider: provider.Name(),
				},
			)
			observation.Fingerprint = observationFingerprint(observation, path)

			return observation, nil
		}
		description, err := provider.inspectDescription(
			ctx,
			path,
			request.Project.ComposerRoot,
		)
		if err != nil {
			return model.ProviderObservation{}, err
		}
		switch description.status {
		case "running":
			observation.State = model.ProviderStateRunning
			runtimeObservation, extensions, err := provider.inspectPHP(
				ctx,
				path,
				request.Project.ComposerRoot,
				description.name,
			)
			if err != nil {
				return model.ProviderObservation{}, err
			}
			composer, err := provider.inspectComposer(
				ctx,
				path,
				request.Project.ComposerRoot,
				description.name,
			)
			if err != nil {
				return model.ProviderObservation{}, err
			}
			observation.Runtimes = append(
				observation.Runtimes,
				runtimeObservation,
			)
			observation.Extensions = append(
				observation.Extensions,
				extensions...,
			)
			observation.Composer = append(observation.Composer, composer)
		case "stopped":
			observation.State = model.ProviderStateStopped
			observation.Runtimes = append(
				observation.Runtimes,
				configuredRuntime(configPath, description.phpVersion),
			)
			observation.Diagnostics = append(
				observation.Diagnostics,
				model.Diagnostic{
					Code:     "ELEFANTE_DDEV_STOPPED",
					Severity: model.SeverityWarning,
					Message:  "The DDEV project is configured but stopped.",
					Hint:     "Review the DDEV start action in elefante plan.",
					Sources: []model.SourceReference{
						{
							Path: configPath,
							Kind: "provider_config",
						},
					},
					Provider: provider.Name(),
				},
			)
		default:
			observation.State = model.ProviderStateDegraded
			observation.Runtimes = append(
				observation.Runtimes,
				configuredRuntime(configPath, description.phpVersion),
			)
			observation.Diagnostics = append(
				observation.Diagnostics,
				model.Diagnostic{
					Code:     "ELEFANTE_DDEV_STATE_UNSUPPORTED",
					Severity: model.SeverityError,
					Message:  "The DDEV project state is not supported.",
					Detail:   "DDEV reported state " + description.status + ".",
					Hint:     "Inspect the DDEV project with ddev describe.",
					Sources: []model.SourceReference{
						{
							Path: configPath,
							Kind: "provider_config",
						},
					},
					Provider: provider.Name(),
				},
			)
		}
		observation.Fingerprint = observationFingerprint(observation, path)

		return observation, nil
	case errors.Is(err, exec.ErrNotFound):
		observation.State = model.ProviderStateUnavailable
		observation.Diagnostics = append(
			observation.Diagnostics,
			model.Diagnostic{
				Code:     "ELEFANTE_DDEV_NOT_FOUND",
				Severity: model.SeverityWarning,
				Message:  "DDEV was not found in the process environment.",
				Hint:     "Install DDEV or select another provider.",
				Sources: []model.SourceReference{
					{
						Path:  "PATH",
						Kind:  "process_environment",
						Field: "ddev",
					},
				},
				Provider: provider.Name(),
			},
		)
		observation.Fingerprint = observationFingerprint(observation, "")

		return observation, nil
	default:
		return model.ProviderObservation{}, providerError(
			"Could not inspect the DDEV executable.",
			err,
		)
	}
}

type ddevDescription struct {
	status     string
	phpVersion string
	name       string
}

func (provider *Provider) inspectDescription(
	ctx context.Context,
	path string,
	projectRoot string,
) (ddevDescription, error) {
	output, err := provider.runner.Output(ctx, executor.Command{
		Executable:       path,
		Arguments:        []string{"describe", "--json-output", "--skip-hooks"},
		WorkingDirectory: projectRoot,
		Environment:      ddevEnvironment(),
	})
	if err != nil {
		return ddevDescription{}, providerError(
			"Could not describe the DDEV project.",
			err,
		)
	}
	var envelope struct {
		Raw struct {
			AppRoot    string `json:"approot"`
			Name       string `json:"name"`
			PHPVersion string `json:"php_version"`
			Status     string `json:"status"`
		} `json:"raw"`
	}
	if err := json.Unmarshal(output, &envelope); err != nil {
		return ddevDescription{}, providerError(
			"DDEV returned an invalid project description.",
			err,
		)
	}
	status := strings.ToLower(strings.TrimSpace(envelope.Raw.Status))
	phpVersion := strings.TrimSpace(envelope.Raw.PHPVersion)
	if strings.TrimSpace(envelope.Raw.AppRoot) == "" ||
		strings.TrimSpace(envelope.Raw.Name) == "" ||
		status == "" ||
		phpVersion == "" {
		return ddevDescription{}, providerError(
			"DDEV returned an incomplete project description.",
			errors.New("project root, state, and PHP version are required"),
		)
	}

	return ddevDescription{
		status:     status,
		phpVersion: phpVersion,
		name:       strings.TrimSpace(envelope.Raw.Name),
	}, nil
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

func configuredRuntime(
	configPath string,
	version string,
) model.RuntimeObservation {
	return model.RuntimeObservation{
		Name:    "php",
		Version: version,
		Source: model.SourceReference{
			Path:  configPath,
			Kind:  "provider_config",
			Field: "php_version",
		},
	}
}

func (provider *Provider) inspectPHP(
	ctx context.Context,
	path string,
	projectRoot string,
	projectName string,
) (
	model.RuntimeObservation,
	[]model.ExtensionObservation,
	error,
) {
	output, err := provider.runner.Output(ctx, ddevExecCommand(
		path,
		projectRoot,
		"php",
		"-r",
		phpInspectionScript,
	))
	if err != nil {
		return model.RuntimeObservation{}, nil, providerError(
			"Could not query PHP inside DDEV.",
			err,
		)
	}

	var inspected phpInspection
	decoder := json.NewDecoder(bytes.NewReader(output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&inspected); err != nil {
		return model.RuntimeObservation{}, nil, providerError(
			"DDEV PHP returned an invalid inspection response.",
			err,
		)
	}
	var trailing struct{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return model.RuntimeObservation{}, nil, providerError(
			"DDEV PHP returned an invalid inspection response.",
			errors.New("inspection response contained trailing data"),
		)
	}
	if strings.TrimSpace(inspected.Version) == "" ||
		strings.TrimSpace(inspected.SAPI) == "" ||
		strings.TrimSpace(inspected.Binary) == "" {
		return model.RuntimeObservation{}, nil, providerError(
			"DDEV PHP returned an incomplete inspection response.",
			errors.New("PHP version, SAPI, and binary are required"),
		)
	}

	source := model.SourceReference{
		Path:  strings.TrimSpace(inspected.Binary),
		Kind:  "provider_executable",
		Field: projectName + ":web",
	}
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

func (provider *Provider) inspectComposer(
	ctx context.Context,
	path string,
	projectRoot string,
	projectName string,
) (model.ComposerObservation, error) {
	pathOutput, err := provider.runner.Output(ctx, ddevExecCommand(
		path,
		projectRoot,
		"which",
		"composer",
	))
	if err != nil {
		return model.ComposerObservation{}, providerError(
			"Could not locate Composer inside DDEV.",
			err,
		)
	}
	composerPath := strings.TrimSpace(string(pathOutput))
	if composerPath == "" {
		return model.ComposerObservation{}, providerError(
			"DDEV returned an empty Composer path.",
			errors.New("Composer path is required"),
		)
	}
	versionOutput, err := provider.runner.Output(ctx, ddevExecCommand(
		path,
		projectRoot,
		"composer",
		"--version",
		"--no-ansi",
		"--no-interaction",
	))
	if err != nil {
		return model.ComposerObservation{}, providerError(
			"Could not query Composer inside DDEV.",
			err,
		)
	}
	version, err := parseComposerVersion(versionOutput)
	if err != nil {
		return model.ComposerObservation{}, providerError(
			"DDEV Composer returned an invalid version response.",
			err,
		)
	}
	source := model.SourceReference{
		Path:  composerPath,
		Kind:  "provider_executable",
		Field: projectName + ":web",
	}

	return model.ComposerObservation{
		Version:   version,
		Source:    "ddev",
		Path:      composerPath,
		Identity:  executableIdentity(projectName, composerPath, version),
		Reference: source,
	}, nil
}

func ddevExecCommand(
	path string,
	projectRoot string,
	executable string,
	arguments ...string,
) executor.Command {
	ddevArguments := []string{
		"--skip-hooks",
		"exec",
		"--raw",
		"--",
		executable,
	}
	ddevArguments = append(ddevArguments, arguments...)

	return executor.Command{
		Executable:       path,
		Arguments:        ddevArguments,
		WorkingDirectory: projectRoot,
		Environment:      ddevEnvironment(),
	}
}

func normalizeExtensionName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.Join(strings.Fields(name), "-")
	if name == "" || strings.HasPrefix(name, "ext-") {
		return name
	}

	return "ext-" + name
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

func executableIdentity(
	projectName string,
	path string,
	version string,
) string {
	sum := sha256.Sum256(
		[]byte(projectName + "\x00" + path + "\x00" + version),
	)

	return "sha256:" + hex.EncodeToString(sum[:])
}

type ddevEngine struct {
	version      string
	platform     string
	environment  string
	architecture string
}

func (provider *Provider) inspectVersion(
	ctx context.Context,
	path string,
) (string, ddevEngine, error) {
	output, err := provider.runner.Output(ctx, executor.Command{
		Executable:  path,
		Arguments:   []string{"version", "--json-output"},
		Environment: ddevEnvironment(),
	})
	if err != nil {
		return "", ddevEngine{}, providerError(
			"Could not query the DDEV version.",
			err,
		)
	}
	var envelope struct {
		Raw map[string]string `json:"raw"`
	}
	if err := json.Unmarshal(output, &envelope); err != nil {
		return "", ddevEngine{}, providerError(
			"DDEV returned an invalid version response.",
			err,
		)
	}
	version := strings.TrimPrefix(
		strings.TrimSpace(envelope.Raw["DDEV version"]),
		"v",
	)
	engine := ddevEngine{
		version:      strings.TrimSpace(envelope.Raw["docker"]),
		platform:     strings.TrimSpace(envelope.Raw["docker-platform"]),
		environment:  strings.TrimSpace(envelope.Raw["ddev-environment"]),
		architecture: strings.TrimSpace(envelope.Raw["architecture"]),
	}
	if version == "" ||
		engine.version == "" ||
		engine.platform == "" ||
		engine.environment == "" ||
		engine.architecture == "" {
		return "", ddevEngine{}, providerError(
			"DDEV returned an incomplete version response.",
			errors.New("required DDEV version fields are missing"),
		)
	}

	return version, engine, nil
}

func (provider *Provider) Plan(
	_ context.Context,
	request providers.ProviderPlanRequest,
) (providers.ProviderPlan, error) {
	result := providers.ProviderPlan{
		Actions:     []model.PlanAction{},
		Diagnostics: []model.Diagnostic{},
	}
	projectRoot := request.Facts.Identity.ComposerRoot
	switch request.Observation.State {
	case model.ProviderStateUnconfigured:
		configPath := filepath.Join(projectRoot, ".ddev", "config.yaml")
		if request.Policy.Frozen {
			result.Diagnostics = append(result.Diagnostics, model.Diagnostic{
				Code:     "ELEFANTE_DDEV_CONFIG_FROZEN",
				Severity: model.SeverityError,
				Message:  "Frozen synchronization cannot create DDEV configuration.",
				Detail:   "The project has no .ddev/config.yaml file.",
				Hint:     "Create and review the DDEV configuration outside frozen mode, then retry.",
				Sources: []model.SourceReference{
					{
						Path: configPath,
						Kind: "provider_config",
					},
				},
				Provider: provider.Name(),
			})

			return result, nil
		}
		result.Actions = append(
			result.Actions,
			model.PlanAction{
				Kind:       model.ActionPrepareProvider,
				Summary:    "Create a minimal DDEV project configuration.",
				Effect:     model.EffectProjectMutation,
				Network:    model.NetworkNone,
				Trust:      model.TrustNone,
				Reversible: true,
				Inputs: []model.ActionInput{
					{
						Name:  "config_path",
						Value: configPath,
					},
					{Name: "operation", Value: "configure"},
					{Name: "project_root", Value: projectRoot},
					{
						Name:  "project_type",
						Value: ddevProjectType(request.Facts.Frameworks),
					},
				},
				ExpectedOutputs: []model.ActionOutput{
					{
						Name:  "configuration",
						Value: configPath,
					},
				},
			},
			startAction(projectRoot),
		)
	case model.ProviderStateStopped:
		result.Actions = append(result.Actions, startAction(projectRoot))
	}

	return result, nil
}

func startAction(projectRoot string) model.PlanAction {
	return model.PlanAction{
		Kind:       model.ActionPrepareProvider,
		Summary:    "Start the DDEV project environment.",
		Effect:     model.EffectProviderMutation,
		Network:    model.NetworkRequired,
		Trust:      model.TrustNone,
		Reversible: true,
		Inputs: []model.ActionInput{
			{Name: "operation", Value: "start"},
			{Name: "project_root", Value: projectRoot},
		},
		ExpectedOutputs: []model.ActionOutput{
			{Name: "provider_state", Value: "running"},
		},
	}
}

func ddevProjectType(frameworks []model.FrameworkFact) string {
	for _, framework := range frameworks {
		if !framework.Primary {
			continue
		}
		switch framework.Kind {
		case model.FrameworkLaravelApplication:
			return "laravel"
		case model.FrameworkBedrockWordPress:
			return "wordpress"
		case model.FrameworkSymfonyApplication:
			return "symfony"
		}
	}

	return "php"
}

func (provider *Provider) Apply(
	context.Context,
	providers.ProviderAction,
	providers.ActionRuntime,
) (providers.ActionResult, error) {
	commandError := model.NewError(
		model.ErrorProvider,
		"DDEV provider actions are not executable yet.",
	)
	commandError.Provider = provider.Name()

	return providers.ActionResult{}, commandError
}

func (provider *Provider) ExecutionSpec(
	_ context.Context,
	request providers.ExecutionRequest,
) (providers.ExecutionSpec, error) {
	if strings.TrimSpace(request.Executable) == "" {
		return providers.ExecutionSpec{}, model.NewError(
			model.ErrorUsage,
			"DDEV execution requires an executable.",
		)
	}
	path, err := provider.runner.LookPath("ddev")
	if err != nil {
		return providers.ExecutionSpec{}, providerError(
			"Could not find the DDEV executable.",
			err,
		)
	}
	arguments := []string{
		"--skip-hooks",
		"exec",
		"--raw",
		"--",
		request.Executable,
	}
	arguments = append(arguments, request.Arguments...)

	return providers.ExecutionSpec{
		Executable:       path,
		Arguments:        arguments,
		WorkingDirectory: request.WorkingDirectory,
		Environment:      append([]string(nil), request.Environment...),
		InputMode:        providers.InputInherit,
		OutputMode:       providers.OutputStream,
	}, nil
}

func emptyObservation() model.ProviderObservation {
	return model.ProviderObservation{
		Provider:     "ddev",
		Engines:      []model.EngineObservation{},
		Capabilities: []model.Capability{},
		Runtimes:     []model.RuntimeObservation{},
		Composer:     []model.ComposerObservation{},
		Extensions:   []model.ExtensionObservation{},
		Diagnostics:  []model.Diagnostic{},
	}
}

func ddevEnvironment() []string {
	return []string{"DDEV_NO_INSTRUMENTATION=true"}
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

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

func providerError(message string, cause error) *model.Error {
	commandError := model.WrapError(model.ErrorProvider, message, cause)
	commandError.Provider = "ddev"
	commandError.Retryable = true

	return commandError
}

func observationFingerprint(
	observation model.ProviderObservation,
	path string,
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
		Executable   string                       `json:"executable,omitempty"`
		Provider     string                       `json:"provider"`
		Available    bool                         `json:"available"`
		Version      string                       `json:"version,omitempty"`
		Platform     string                       `json:"platform,omitempty"`
		Architecture string                       `json:"architecture,omitempty"`
		State        model.ProviderState          `json:"state,omitempty"`
		Engines      []model.EngineObservation    `json:"engines"`
		Capabilities []model.Capability           `json:"capabilities"`
		Runtimes     []model.RuntimeObservation   `json:"runtimes"`
		Composer     []model.ComposerObservation  `json:"composer"`
		Extensions   []model.ExtensionObservation `json:"extensions"`
		Diagnostics  []stableDiagnostic           `json:"diagnostics"`
	}{
		Schema:       "elefante.provider-observation/v1",
		Executable:   path,
		Provider:     observation.Provider,
		Available:    observation.Available,
		Version:      observation.Version,
		Platform:     observation.Platform,
		Architecture: observation.Architecture,
		State:        observation.State,
		Engines: append(
			[]model.EngineObservation(nil),
			observation.Engines...,
		),
		Capabilities: append(
			[]model.Capability(nil),
			observation.Capabilities...,
		),
		Runtimes:   append([]model.RuntimeObservation(nil), observation.Runtimes...),
		Composer:   append([]model.ComposerObservation(nil), observation.Composer...),
		Extensions: append([]model.ExtensionObservation(nil), observation.Extensions...),
	}
	for _, diagnostic := range observation.Diagnostics {
		canonical.Diagnostics = append(
			canonical.Diagnostics,
			stableDiagnostic{
				Code:     diagnostic.Code,
				Severity: diagnostic.Severity,
				Sources: append(
					[]model.SourceReference(nil),
					diagnostic.Sources...,
				),
				Provider:  diagnostic.Provider,
				Retryable: diagnostic.Retryable,
			},
		)
	}
	encoded, _ := json.Marshal(canonical)
	sum := sha256.Sum256(encoded)

	return "sha256:" + hex.EncodeToString(sum[:])
}
