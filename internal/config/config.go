package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/elefantephp/elefante/internal/model"
	projectpaths "github.com/elefantephp/elefante/internal/paths"
	"github.com/pelletier/go-toml/v2"
)

type Document struct {
	Path           string
	RepositoryRoot string
	Content        []byte
}

type Result struct {
	Facts       model.ConfigFacts
	Diagnostics []model.Diagnostic
}

const MaxDocumentSize = 64 * 1024

type documentSchema struct {
	SchemaVersion *int                  `toml:"schema_version"`
	Project       projectSchema         `toml:"project"`
	Providers     providerSchema        `toml:"providers"`
	Composer      composerSchema        `toml:"composer"`
	Extensions    extensionSchema       `toml:"extensions"`
	Tasks         map[string]taskSchema `toml:"tasks"`
	CI            ciSchema              `toml:"ci"`
}

type projectSchema struct {
	ComposerRoot string `toml:"composer_root"`
}

type providerSchema struct {
	Preferred []string `toml:"preferred"`
	Allowed   []string `toml:"allowed"`
	Denied    []string `toml:"denied"`
}

type composerSchema struct {
	Constraint string `toml:"constraint"`
}

type extensionSchema struct {
	Optional []string `toml:"optional"`
}

type taskSchema struct {
	Command          []string `toml:"command"`
	WorkingDirectory string   `toml:"working_directory"`
}

type ciSchema struct {
	Provider string `toml:"provider"`
	Frozen   bool   `toml:"frozen"`
}

var primaryCommands = map[string]struct{}{
	"doctor": {},
	"plan":   {},
	"run":    {},
	"sync":   {},
	"tool":   {},
}

func Parse(document Document) Result {
	result := Result{
		Facts: model.ConfigFacts{Path: document.Path},
	}
	if len(document.Content) > MaxDocumentSize {
		result.Diagnostics = append(
			result.Diagnostics,
			configDiagnostic(
				document.Path,
				"ELEFANTE_CONFIG_TOO_LARGE",
				"Elefante configuration exceeds the supported size limit.",
				fmt.Sprintf(
					"The file is %d bytes and the limit is %d bytes.",
					len(document.Content),
					MaxDocumentSize,
				),
				0,
			),
		)

		return result
	}
	var raw map[string]any
	if err := toml.Unmarshal(document.Content, &raw); err == nil &&
		containsProhibitedSecret(raw) {
		result.Diagnostics = append(
			result.Diagnostics,
			configDiagnostic(
				document.Path,
				"ELEFANTE_CONFIG_SECRET_VALUE",
				"Elefante configuration cannot contain secret values.",
				"Commit environment variable names or provider policy, never credentials.",
				0,
			),
		)

		return result
	}
	var decoded documentSchema
	decoder := toml.NewDecoder(bytes.NewReader(document.Content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		var strictMissing *toml.StrictMissingError
		if errors.As(err, &strictMissing) {
			result.Diagnostics = append(
				result.Diagnostics,
				configWarning(
					document.Path,
					"ELEFANTE_CONFIG_UNKNOWN_FIELD",
					"Elefante configuration contains an unknown field.",
					"Remove fields that are not part of schema version 1.",
					decodeErrorLine(err),
				),
			)

			return result
		}
		result.Diagnostics = append(
			result.Diagnostics,
			configDiagnostic(
				document.Path,
				"ELEFANTE_CONFIG_INVALID",
				"Elefante configuration is not valid version 1 TOML.",
				err.Error(),
				decodeErrorLine(err),
			),
		)

		return result
	}
	if decoded.SchemaVersion == nil {
		result.Diagnostics = append(
			result.Diagnostics,
			configDiagnostic(
				document.Path,
				"ELEFANTE_CONFIG_SCHEMA_REQUIRED",
				"Elefante configuration must declare schema_version = 1.",
				"The schema_version field is missing.",
				0,
			),
		)

		return result
	}
	if *decoded.SchemaVersion != 1 {
		result.Diagnostics = append(
			result.Diagnostics,
			configDiagnostic(
				document.Path,
				"ELEFANTE_CONFIG_SCHEMA_UNSUPPORTED",
				"Elefante configuration must declare schema_version = 1.",
				fmt.Sprintf("Found schema version %d.", *decoded.SchemaVersion),
				0,
			),
		)

		return result
	}

	composerRoot, err := normalizePolicyPath(
		filepath.Dir(document.Path),
		defaultPath(decoded.Project.ComposerRoot),
		document.RepositoryRoot,
	)
	if err != nil {
		result.Diagnostics = append(
			result.Diagnostics,
			configDiagnostic(
				document.Path,
				"ELEFANTE_CONFIG_PATH_INVALID",
				"Elefante configuration contains an invalid project path.",
				err.Error(),
				0,
			),
		)

		return result
	}

	result.Facts = model.ConfigFacts{
		Path:          document.Path,
		SchemaVersion: *decoded.SchemaVersion,
		Project: model.ConfigProjectPolicy{
			ComposerRoot: composerRoot,
		},
		Providers: model.ConfigProviderPolicy{
			Preferred: normalizeOrderedNames(decoded.Providers.Preferred),
			Allowed:   normalizeNames(decoded.Providers.Allowed),
			Denied:    normalizeNames(decoded.Providers.Denied),
		},
		Composer: model.ConfigComposerPolicy{
			Constraint: strings.TrimSpace(decoded.Composer.Constraint),
		},
		Extensions: model.ConfigExtensionPolicy{
			Optional: normalizeNames(decoded.Extensions.Optional),
		},
		CI: model.ConfigCIPolicy{
			Provider: strings.ToLower(strings.TrimSpace(decoded.CI.Provider)),
			Frozen:   decoded.CI.Frozen,
		},
	}

	taskNames := make([]string, 0, len(decoded.Tasks))
	for name := range decoded.Tasks {
		taskNames = append(taskNames, name)
	}
	sort.Strings(taskNames)
	for _, name := range taskNames {
		task := decoded.Tasks[name]
		if commandContainsProhibitedSecret(task.Command) {
			result.Diagnostics = append(
				result.Diagnostics,
				configDiagnostic(
					document.Path,
					"ELEFANTE_CONFIG_SECRET_VALUE",
					fmt.Sprintf("Task %q cannot contain secret values.", name),
					"Pass credentials through the selected provider environment at execution time.",
					0,
				),
			)

			return result
		}
		if _, shadowsPrimaryCommand := primaryCommands[name]; shadowsPrimaryCommand {
			result.Diagnostics = append(
				result.Diagnostics,
				configDiagnostic(
					document.Path,
					"ELEFANTE_CONFIG_TASK_SHADOW",
					fmt.Sprintf("Task %q shadows a primary Elefante command.", name),
					"Choose a task name that does not match doctor, plan, sync, run, or tool.",
					0,
				),
			)

			return result
		}
		if len(task.Command) == 0 || strings.TrimSpace(task.Command[0]) == "" {
			result.Diagnostics = append(
				result.Diagnostics,
				configDiagnostic(
					document.Path,
					"ELEFANTE_CONFIG_TASK_INVALID",
					fmt.Sprintf("Task %q must declare an executable argument.", name),
					"Use a nonempty TOML array whose first value is the executable.",
					0,
				),
			)

			return result
		}
		workingDirectory, err := normalizePolicyPath(
			filepath.Dir(document.Path),
			defaultPath(task.WorkingDirectory),
			document.RepositoryRoot,
		)
		if err != nil {
			result.Diagnostics = append(
				result.Diagnostics,
				configDiagnostic(
					document.Path,
					"ELEFANTE_CONFIG_PATH_INVALID",
					fmt.Sprintf("Task %q contains an invalid working directory.", name),
					err.Error(),
					0,
				),
			)

			return result
		}
		result.Facts.Tasks = append(result.Facts.Tasks, model.ConfigTask{
			Name:             name,
			Command:          cloneStrings(task.Command),
			WorkingDirectory: workingDirectory,
		})
	}

	return result
}

func normalizePolicyPath(
	base string,
	value string,
	repositoryRoot string,
) (string, error) {
	path := value
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	path = filepath.Clean(path)
	resolvedPath, err := resolveExistingPrefix(path)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", path, err)
	}
	resolvedRepositoryRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return "", fmt.Errorf("resolve repository %q: %w", repositoryRoot, err)
	}
	if !projectpaths.Contains(resolvedRepositoryRoot, resolvedPath) {
		return "", fmt.Errorf(
			"path %q escapes repository %q",
			resolvedPath,
			resolvedRepositoryRoot,
		)
	}

	return resolvedPath, nil
}

func resolveExistingPrefix(path string) (string, error) {
	candidate := filepath.Clean(path)
	for {
		_, err := os.Lstat(candidate)
		switch {
		case err == nil:
			resolved, err := filepath.EvalSymlinks(candidate)
			if err != nil {
				return "", err
			}
			remainder, err := filepath.Rel(candidate, path)
			if err != nil {
				return "", err
			}

			return filepath.Clean(filepath.Join(resolved, remainder)), nil
		case !errors.Is(err, os.ErrNotExist):
			return "", err
		}

		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", fmt.Errorf("no existing path boundary")
		}
		candidate = parent
	}
}

func configDiagnostic(
	path string,
	code string,
	message string,
	detail string,
	line int,
) model.Diagnostic {
	return model.Diagnostic{
		Code:     code,
		Severity: model.SeverityError,
		Message:  message,
		Detail:   detail,
		Hint:     "Correct elefante.toml before synchronizing the project.",
		Sources: []model.SourceReference{
			{
				Path: path,
				Kind: "elefante_config",
				Line: line,
			},
		},
	}
}

func configWarning(
	path string,
	code string,
	message string,
	detail string,
	line int,
) model.Diagnostic {
	diagnostic := configDiagnostic(path, code, message, detail, line)
	diagnostic.Severity = model.SeverityWarning

	return diagnostic
}

func decodeErrorLine(err error) int {
	var decodeError *toml.DecodeError
	if errors.As(err, &decodeError) {
		line, _ := decodeError.Position()

		return line
	}

	return 0
}

func defaultPath(value string) string {
	if value == "" {
		return "."
	}

	return value
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func normalizeNames(values []string) []string {
	normalized := normalizeOrderedNames(values)
	sort.Strings(normalized)

	return normalized
}

func normalizeOrderedNames(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	return normalized
}

func containsProhibitedSecret(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isSensitiveName(key) && hasCommittedValue(child) {
				return true
			}
			if containsProhibitedSecret(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsProhibitedSecret(child) {
				return true
			}
		}
	}

	return false
}

func hasCommittedValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func isSensitiveName(name string) bool {
	normalized := strings.NewReplacer("-", "_", ".", "_").
		Replace(strings.ToLower(strings.TrimSpace(name)))
	sensitiveNames := []string{
		"api_key",
		"authorization",
		"cookie",
		"credential",
		"credentials",
		"password",
		"passwd",
		"private_key",
		"secret",
		"token",
	}
	for _, sensitive := range sensitiveNames {
		if normalized == sensitive ||
			strings.HasPrefix(normalized, sensitive+"_") ||
			strings.HasSuffix(normalized, "_"+sensitive) {
			return true
		}
	}

	return false
}

func commandContainsProhibitedSecret(command []string) bool {
	for index, argument := range command {
		normalized := strings.ToLower(strings.TrimSpace(argument))
		if key, value, found := strings.Cut(normalized, "="); found &&
			isSensitiveName(strings.TrimLeft(key, "-")) &&
			value != "" {
			return true
		}
		if !strings.HasPrefix(normalized, "--") ||
			!isSensitiveName(strings.TrimLeft(normalized, "-")) {
			continue
		}
		if index+1 < len(command) && command[index+1] != "" {
			return true
		}
	}

	return false
}
