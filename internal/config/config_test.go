package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/config"
)

func TestParseNormalizesVersionOnePolicy(t *testing.T) {
	repositoryRoot := t.TempDir()
	composerRoot := filepath.Join(repositoryRoot, "apps", "web")
	if err := os.MkdirAll(composerRoot, 0o755); err != nil {
		t.Fatalf("create Composer root: %v", err)
	}
	resolvedComposerRoot, err := filepath.EvalSymlinks(composerRoot)
	if err != nil {
		t.Fatalf("resolve Composer root: %v", err)
	}
	configPath := filepath.Join(repositoryRoot, "elefante.toml")
	content := []byte(`
schema_version = 1

[project]
composer_root = "apps/web"

[providers]
preferred = ["native", "ddev"]
allowed = ["native", "ddev", "homebrew"]
denied = []

[composer]
constraint = "^2"

[extensions]
optional = ["ext-xdebug"]

[tasks.build]
command = ["php", "artisan", "optimize"]
working_directory = "apps/web"

[tasks.test]
command = ["php", "artisan", "test"]
working_directory = "apps/web"

[ci]
provider = "ddev"
frozen = true
`)

	result := config.Parse(config.Document{
		Path:           configPath,
		RepositoryRoot: repositoryRoot,
		Content:        content,
	})

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected valid configuration, got %#v", result.Diagnostics)
	}
	if result.Facts.Path != configPath {
		t.Fatalf("expected config path %q, got %q", configPath, result.Facts.Path)
	}
	if result.Facts.SchemaVersion != 1 {
		t.Fatalf("expected schema version 1, got %d", result.Facts.SchemaVersion)
	}
	if result.Facts.Project.ComposerRoot != resolvedComposerRoot {
		t.Fatalf(
			"expected normalized Composer root %q, got %q",
			resolvedComposerRoot,
			result.Facts.Project.ComposerRoot,
		)
	}
	if !reflect.DeepEqual(
		result.Facts.Providers.Preferred,
		[]string{"native", "ddev"},
	) {
		t.Fatalf("unexpected preferred providers %#v", result.Facts.Providers.Preferred)
	}
	if result.Facts.Composer.Constraint != "^2" {
		t.Fatalf("unexpected Composer constraint %q", result.Facts.Composer.Constraint)
	}
	if !reflect.DeepEqual(
		result.Facts.Extensions.Optional,
		[]string{"ext-xdebug"},
	) {
		t.Fatalf("unexpected optional extensions %#v", result.Facts.Extensions.Optional)
	}

	expectedTasks := []struct {
		name    string
		command []string
	}{
		{name: "build", command: []string{"php", "artisan", "optimize"}},
		{name: "test", command: []string{"php", "artisan", "test"}},
	}
	if len(result.Facts.Tasks) != len(expectedTasks) {
		t.Fatalf("expected two normalized tasks, got %#v", result.Facts.Tasks)
	}
	for index, expected := range expectedTasks {
		task := result.Facts.Tasks[index]
		if task.Name != expected.name {
			t.Fatalf("expected task %q, got %q", expected.name, task.Name)
		}
		if !reflect.DeepEqual(task.Command, expected.command) {
			t.Fatalf("expected argument vector %#v, got %#v", expected.command, task.Command)
		}
		if task.WorkingDirectory != resolvedComposerRoot {
			t.Fatalf(
				"expected normalized task directory %q, got %q",
				resolvedComposerRoot,
				task.WorkingDirectory,
			)
		}
	}
	if result.Facts.CI.Provider != "ddev" || !result.Facts.CI.Frozen {
		t.Fatalf("unexpected CI policy %#v", result.Facts.CI)
	}
}

func TestParseRequiresSupportedSchemaAndRejectsUnknownFields(t *testing.T) {
	repositoryRoot := t.TempDir()
	configPath := filepath.Join(repositoryRoot, "elefante.toml")

	tests := []struct {
		name string
		toml string
		code string
	}{
		{
			name: "missing schema version",
			toml: "[project]\ncomposer_root = \".\"\n",
			code: "ELEFANTE_CONFIG_SCHEMA_REQUIRED",
		},
		{
			name: "unsupported schema version",
			toml: "schema_version = 2\n",
			code: "ELEFANTE_CONFIG_SCHEMA_UNSUPPORTED",
		},
		{
			name: "unknown top level field",
			toml: "schema_version = 1\nsurprise = true\n",
			code: "ELEFANTE_CONFIG_UNKNOWN_FIELD",
		},
		{
			name: "unknown nested field",
			toml: "schema_version = 1\n[providers]\nsurprise = []\n",
			code: "ELEFANTE_CONFIG_UNKNOWN_FIELD",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := config.Parse(config.Document{
				Path:           configPath,
				RepositoryRoot: repositoryRoot,
				Content:        []byte(test.toml),
			})

			if len(result.Diagnostics) != 1 {
				t.Fatalf("expected one diagnostic, got %#v", result.Diagnostics)
			}
			if result.Diagnostics[0].Code != test.code {
				t.Fatalf(
					"expected diagnostic %q, got %#v",
					test.code,
					result.Diagnostics[0],
				)
			}
			if strings.Contains(test.name, "unknown") &&
				result.Diagnostics[0].Severity != "warning" {
				t.Fatalf(
					"expected unknown field warning, got %#v",
					result.Diagnostics[0],
				)
			}
			if len(result.Diagnostics[0].Sources) != 1 {
				t.Fatalf("expected one source, got %#v", result.Diagnostics[0].Sources)
			}
			if result.Diagnostics[0].Sources[0].Path != configPath {
				t.Fatalf(
					"expected source path %q, got %#v",
					configPath,
					result.Diagnostics[0].Sources[0],
				)
			}
		})
	}
}

func TestParseRejectsUnsafeTaskDefinitions(t *testing.T) {
	repositoryRoot := t.TempDir()
	configPath := filepath.Join(repositoryRoot, "elefante.toml")

	tests := []struct {
		name string
		toml string
		code string
	}{
		{
			name: "shell command string",
			toml: `
schema_version = 1
[tasks.test]
command = "php artisan test"
`,
			code: "ELEFANTE_CONFIG_INVALID",
		},
		{
			name: "empty argument vector",
			toml: `
schema_version = 1
[tasks.test]
command = []
`,
			code: "ELEFANTE_CONFIG_TASK_INVALID",
		},
		{
			name: "empty executable",
			toml: `
schema_version = 1
[tasks.test]
command = ["", "argument"]
`,
			code: "ELEFANTE_CONFIG_TASK_INVALID",
		},
		{
			name: "primary command shadow",
			toml: `
schema_version = 1
[tasks.doctor]
command = ["php", "doctor.php"]
`,
			code: "ELEFANTE_CONFIG_TASK_SHADOW",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := config.Parse(config.Document{
				Path:           configPath,
				RepositoryRoot: repositoryRoot,
				Content:        []byte(test.toml),
			})

			if len(result.Diagnostics) != 1 {
				t.Fatalf("expected one diagnostic, got %#v", result.Diagnostics)
			}
			if result.Diagnostics[0].Code != test.code {
				t.Fatalf(
					"expected diagnostic %q, got %#v",
					test.code,
					result.Diagnostics[0],
				)
			}
		})
	}
}

func TestParsePreservesTaskArgumentsWithoutShellInterpretation(t *testing.T) {
	repositoryRoot := t.TempDir()
	configPath := filepath.Join(repositoryRoot, "elefante.toml")
	content := []byte(`
schema_version = 1
[tasks.test]
command = ["php", "script.php", "value with spaces", "$(touch should-not-exist)"]
`)

	result := config.Parse(config.Document{
		Path:           configPath,
		RepositoryRoot: repositoryRoot,
		Content:        content,
	})

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected valid configuration, got %#v", result.Diagnostics)
	}
	expected := []string{
		"php",
		"script.php",
		"value with spaces",
		"$(touch should-not-exist)",
	}
	if len(result.Facts.Tasks) != 1 {
		t.Fatalf("expected one task, got %#v", result.Facts.Tasks)
	}
	if !reflect.DeepEqual(result.Facts.Tasks[0].Command, expected) {
		t.Fatalf(
			"expected exact argument vector %#v, got %#v",
			expected,
			result.Facts.Tasks[0].Command,
		)
	}
}

func TestParseRejectsPolicyPathsOutsideRepository(t *testing.T) {
	repositoryRoot := t.TempDir()
	outsideRoot := t.TempDir()
	configPath := filepath.Join(repositoryRoot, "elefante.toml")
	escapeLink := filepath.Join(repositoryRoot, "outside-link")
	if err := os.Symlink(outsideRoot, escapeLink); err != nil {
		t.Fatalf("create escape symlink: %v", err)
	}

	tests := []struct {
		name string
		toml string
	}{
		{
			name: "Composer root parent escape",
			toml: `
schema_version = 1
[project]
composer_root = "../outside"
`,
		},
		{
			name: "task working directory parent escape",
			toml: `
schema_version = 1
[tasks.test]
command = ["php", "artisan", "test"]
working_directory = "../outside"
`,
		},
		{
			name: "Composer root symlink escape",
			toml: `
schema_version = 1
[project]
composer_root = "outside-link"
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := config.Parse(config.Document{
				Path:           configPath,
				RepositoryRoot: repositoryRoot,
				Content:        []byte(test.toml),
			})

			if len(result.Diagnostics) != 1 {
				t.Fatalf("expected one diagnostic, got %#v", result.Diagnostics)
			}
			if result.Diagnostics[0].Code != "ELEFANTE_CONFIG_PATH_INVALID" {
				t.Fatalf("expected path diagnostic, got %#v", result.Diagnostics[0])
			}
		})
	}
}

func TestParseRejectsProhibitedSecretValuesWithoutEchoingThem(t *testing.T) {
	repositoryRoot := t.TempDir()
	configPath := filepath.Join(repositoryRoot, "elefante.toml")
	const syntheticSecret = "elefante-synthetic-secret-4381"

	tests := []struct {
		name string
		toml string
	}{
		{
			name: "secret shaped configuration field",
			toml: `
schema_version = 1
[environment]
APP_SECRET = "` + syntheticSecret + `"
`,
		},
		{
			name: "secret task argument",
			toml: `
schema_version = 1
[tasks.deploy]
command = ["deploy", "--token", "` + syntheticSecret + `"]
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := config.Parse(config.Document{
				Path:           configPath,
				RepositoryRoot: repositoryRoot,
				Content:        []byte(test.toml),
			})

			if len(result.Diagnostics) != 1 {
				t.Fatalf("expected one diagnostic, got %#v", result.Diagnostics)
			}
			diagnostic := result.Diagnostics[0]
			if diagnostic.Code != "ELEFANTE_CONFIG_SECRET_VALUE" {
				t.Fatalf("expected secret diagnostic, got %#v", diagnostic)
			}
			if strings.Contains(
				diagnostic.Message+diagnostic.Detail+diagnostic.Hint,
				syntheticSecret,
			) {
				t.Fatalf("diagnostic exposed the synthetic secret: %#v", diagnostic)
			}
		})
	}
}

func TestParseNormalizesProviderAndExtensionNames(t *testing.T) {
	repositoryRoot := t.TempDir()
	configPath := filepath.Join(repositoryRoot, "elefante.toml")
	content := []byte(`
schema_version = 1
[providers]
preferred = [" Native ", "DDEV", "native"]
allowed = ["homebrew", "DDEV", "native", "ddev"]
denied = ["HERD", "herd"]
[extensions]
optional = [" EXT-XDEBUG ", "ext-redis", "ext-xdebug"]
[ci]
provider = " DDEV "
`)

	result := config.Parse(config.Document{
		Path:           configPath,
		RepositoryRoot: repositoryRoot,
		Content:        content,
	})

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected valid configuration, got %#v", result.Diagnostics)
	}
	if !reflect.DeepEqual(
		result.Facts.Providers.Preferred,
		[]string{"native", "ddev"},
	) {
		t.Fatalf("unexpected preferred providers %#v", result.Facts.Providers.Preferred)
	}
	if !reflect.DeepEqual(
		result.Facts.Providers.Allowed,
		[]string{"ddev", "homebrew", "native"},
	) {
		t.Fatalf("unexpected allowed providers %#v", result.Facts.Providers.Allowed)
	}
	if !reflect.DeepEqual(result.Facts.Providers.Denied, []string{"herd"}) {
		t.Fatalf("unexpected denied providers %#v", result.Facts.Providers.Denied)
	}
	if !reflect.DeepEqual(
		result.Facts.Extensions.Optional,
		[]string{"ext-redis", "ext-xdebug"},
	) {
		t.Fatalf("unexpected optional extensions %#v", result.Facts.Extensions.Optional)
	}
	if result.Facts.CI.Provider != "ddev" {
		t.Fatalf("unexpected CI provider %q", result.Facts.CI.Provider)
	}
}

func TestParseRejectsOversizedConfigurationBeforeTOMLDecoding(t *testing.T) {
	repositoryRoot := t.TempDir()
	configPath := filepath.Join(repositoryRoot, "elefante.toml")
	content := make([]byte, config.MaxDocumentSize+1)

	result := config.Parse(config.Document{
		Path:           configPath,
		RepositoryRoot: repositoryRoot,
		Content:        content,
	})

	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "ELEFANTE_CONFIG_TOO_LARGE" {
		t.Fatalf("expected size diagnostic, got %#v", result.Diagnostics[0])
	}
}
