package composer_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/model"
)

func TestParseProjectCapturesRootRequirementsWithSources(t *testing.T) {
	manifest := readFixture(t, "root-requirements", "composer.json")
	manifestPath := "/workspace/composer.json"

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    manifestPath,
			Content: manifest,
		},
	})
	if err != nil {
		t.Fatalf("parse Composer project: %v", err)
	}

	expectedLinks := []model.ComposerLink{
		{
			Name:       "composer",
			Constraint: "^2.7",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/require/Composer",
			},
		},
		{
			Name:       "composer-runtime-api",
			Constraint: "^2.2",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/require/composer-runtime-api",
			},
		},
		{
			Name:       "ext-intl",
			Constraint: "*",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/require/EXT-Intl",
			},
		},
		{
			Name:       "laravel/framework",
			Constraint: "^12.0",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/require/laravel~1framework",
			},
		},
		{
			Name:       "lib-icu",
			Constraint: ">=73",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/require/lib-icu",
			},
		},
		{
			Name:       "php",
			Constraint: "^8.3 || ^8.4",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/require/PHP",
			},
		},
		{
			Name:       "php-64bit",
			Constraint: "*",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/require/PHP-64BIT",
			},
		},
	}
	assertComposerLinks(t, result.Facts.Manifest.Requirements, expectedLinks)

	expectedPlatform := []model.Requirement{
		{
			Name:       "composer",
			Kind:       model.RequirementComposer,
			Constraint: "^2.7",
			Scope:      model.RequirementScopeRoot,
			Sources:    []model.SourceReference{expectedLinks[0].Source},
		},
		{
			Name:       "composer-runtime-api",
			Kind:       model.RequirementComposerRuntimeAPI,
			Constraint: "^2.2",
			Scope:      model.RequirementScopeRoot,
			Sources:    []model.SourceReference{expectedLinks[1].Source},
		},
		{
			Name:       "ext-intl",
			Kind:       model.RequirementExtension,
			Constraint: "*",
			Scope:      model.RequirementScopeRoot,
			Sources:    []model.SourceReference{expectedLinks[2].Source},
		},
		{
			Name:       "lib-icu",
			Kind:       model.RequirementSystemLibrary,
			Constraint: ">=73",
			Scope:      model.RequirementScopeRoot,
			Sources:    []model.SourceReference{expectedLinks[4].Source},
		},
		{
			Name:       "php",
			Kind:       model.RequirementPHP,
			Constraint: "^8.3 || ^8.4",
			Scope:      model.RequirementScopeRoot,
			Sources:    []model.SourceReference{expectedLinks[5].Source},
		},
		{
			Name:       "php-64bit",
			Kind:       model.RequirementPHPSubtype,
			Constraint: "*",
			Scope:      model.RequirementScopeRoot,
			Sources:    []model.SourceReference{expectedLinks[6].Source},
		},
	}
	if len(result.Facts.PlatformRequirements) != len(expectedPlatform) {
		t.Fatalf(
			"expected %d platform requirements, got %#v",
			len(expectedPlatform),
			result.Facts.PlatformRequirements,
		)
	}
	for index, expected := range expectedPlatform {
		if got := result.Facts.PlatformRequirements[index]; !equalRequirement(got, expected) {
			t.Errorf("platform requirement %d\nexpected: %#v\ngot:      %#v", index, expected, got)
		}
	}
}

func TestParseProjectSeparatesDevelopmentRequirementsAndConflicts(t *testing.T) {
	manifest := readFixture(t, "development-conflicts", "composer.json")
	manifestPath := "/workspace/composer.json"

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    manifestPath,
			Content: manifest,
		},
	})
	if err != nil {
		t.Fatalf("parse Composer project: %v", err)
	}

	expectedDevelopment := []model.ComposerLink{
		{
			Name:       "ext-xdebug",
			Constraint: "^3.3",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/require-dev/ext-xdebug",
			},
		},
		{
			Name:       "phpunit/phpunit",
			Constraint: "^12.0",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/require-dev/phpunit~1phpunit",
			},
		},
	}
	assertComposerLinks(
		t,
		result.Facts.Manifest.DevelopmentRequirements,
		expectedDevelopment,
	)

	expectedConflicts := []model.ComposerLink{
		{
			Name:       "ext-mysql",
			Constraint: "*",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/conflict/ext-mysql",
			},
		},
		{
			Name:       "php",
			Constraint: "<8.3",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/conflict/php",
			},
		},
	}
	assertComposerLinks(t, result.Facts.Manifest.Conflicts, expectedConflicts)

	expectedScopes := []model.RequirementScope{
		model.RequirementScopeRootDevelopment,
		model.RequirementScopeRootConflict,
		model.RequirementScopeRootConflict,
	}
	if len(result.Facts.PlatformRequirements) != len(expectedScopes) {
		t.Fatalf(
			"expected %d platform facts, got %#v",
			len(expectedScopes),
			result.Facts.PlatformRequirements,
		)
	}
	for index, expectedScope := range expectedScopes {
		if got := result.Facts.PlatformRequirements[index].Scope; got != expectedScope {
			t.Errorf("platform fact %d expected scope %q, got %q", index, expectedScope, got)
		}
	}
}

func TestParseProjectKeepsPlatformEmulationSeparateFromRequirements(t *testing.T) {
	manifest := readFixture(t, "platform-emulation", "composer.json")
	manifestPath := "/workspace/composer.json"

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    manifestPath,
			Content: manifest,
		},
	})
	if err != nil {
		t.Fatalf("parse Composer project: %v", err)
	}

	if len(result.Facts.PlatformRequirements) != 0 {
		t.Fatalf(
			"expected no actual platform requirements, got %#v",
			result.Facts.PlatformRequirements,
		)
	}

	expected := []model.PlatformOverride{
		{
			Name:     "ext-redis",
			Kind:     model.RequirementExtension,
			Disabled: true,
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/config/platform/EXT-Redis",
			},
		},
		{
			Name:    "lib-curl",
			Kind:    model.RequirementSystemLibrary,
			Version: "8.1.0",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/config/platform/lib-curl",
			},
		},
		{
			Name:    "php",
			Kind:    model.RequirementPHP,
			Version: "8.3.12",
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/config/platform/PHP",
			},
		},
	}

	if len(result.Facts.PlatformEmulation) != len(expected) {
		t.Fatalf(
			"expected %d platform overrides, got %#v",
			len(expected),
			result.Facts.PlatformEmulation,
		)
	}
	for index := range expected {
		if result.Facts.PlatformEmulation[index] != expected[index] {
			t.Errorf(
				"platform override %d\nexpected: %#v\ngot:      %#v",
				index,
				expected[index],
				result.Facts.PlatformEmulation[index],
			)
		}
	}
}

func TestParseProjectDiscoversScriptsWithoutExposingCommands(t *testing.T) {
	manifest := readFixture(t, "scripts", "composer.json")
	manifestPath := "/workspace/composer.json"

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    manifestPath,
			Content: manifest,
		},
	})
	if err != nil {
		t.Fatalf("parse Composer project: %v", err)
	}

	expected := []model.ComposerScript{
		{
			Name:          "post-install-cmd",
			CommandCount:  1,
			ContentSHA256: scriptDigest(t, []string{"Acme\\Setup::run"}),
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/scripts/post-install-cmd",
			},
		},
		{
			Name:         "secret-check",
			CommandCount: 1,
			ContentSHA256: scriptDigest(t, []string{
				"curl https://example.invalid/?token=synthetic-secret-value",
			}),
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/scripts/secret-check",
			},
		},
		{
			Name:         "test",
			CommandCount: 2,
			ContentSHA256: scriptDigest(t, []string{
				"@php vendor/bin/phpunit",
				"@composer validate",
			}),
			Source: model.SourceReference{
				Path:  manifestPath,
				Kind:  "composer_manifest",
				Field: "/scripts/test",
			},
		},
	}

	if len(result.Facts.Scripts) != len(expected) {
		t.Fatalf("expected %d scripts, got %#v", len(expected), result.Facts.Scripts)
	}
	for index := range expected {
		if result.Facts.Scripts[index] != expected[index] {
			t.Errorf(
				"script %d\nexpected: %#v\ngot:      %#v",
				index,
				expected[index],
				result.Facts.Scripts[index],
			)
		}
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal parse result: %v", err)
	}
	if strings.Contains(string(encoded), "synthetic-secret-value") {
		t.Fatalf("expected script command content to stay out of facts, got %s", encoded)
	}
}

func TestParseProjectReportsMissingLockFile(t *testing.T) {
	manifest := readFixture(t, "root-requirements", "composer.json")
	manifestPath := "/workspace/composer.json"

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    manifestPath,
			Content: manifest,
		},
	})
	if err != nil {
		t.Fatalf("parse Composer project: %v", err)
	}

	if result.Facts.Lock.Status != model.ComposerLockMissing {
		t.Errorf("expected missing lock status, got %q", result.Facts.Lock.Status)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one missing lock diagnostic, got %#v", result.Diagnostics)
	}

	diagnostic := result.Diagnostics[0]
	if diagnostic.Code != "ELEFANTE_COMPOSER_LOCK_MISSING" {
		t.Errorf("unexpected diagnostic code %q", diagnostic.Code)
	}
	if diagnostic.Severity != model.SeverityWarning {
		t.Errorf("expected warning severity, got %q", diagnostic.Severity)
	}
	if len(diagnostic.Sources) != 1 {
		t.Fatalf("expected one diagnostic source, got %#v", diagnostic.Sources)
	}
	expectedSource := model.SourceReference{
		Path: "/workspace/composer.lock",
		Kind: "composer_lock",
	}
	if diagnostic.Sources[0] != expectedSource {
		t.Errorf("expected source %#v, got %#v", expectedSource, diagnostic.Sources[0])
	}
}

func TestParseProjectValidatesFreshLockAndCapturesLockedPlatformRequirements(t *testing.T) {
	manifest := readFixture(t, "locked-platform", "composer.json")
	lock := readFixture(t, "locked-platform", "composer.lock")
	manifestPath := "/workspace/composer.json"
	lockPath := "/workspace/composer.lock"

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    manifestPath,
			Content: manifest,
		},
		Lock: &composer.Document{
			Path:    lockPath,
			Content: lock,
		},
	})
	if err != nil {
		t.Fatalf("parse Composer project: %v", err)
	}

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no lock diagnostics, got %#v", result.Diagnostics)
	}
	if result.Facts.Lock.Status != model.ComposerLockFresh {
		t.Errorf("expected fresh lock status, got %q", result.Facts.Lock.Status)
	}
	if result.Facts.Lock.ContentHash != "8c785f5fae6d3a41bcb172e2b2bcd347" {
		t.Errorf("unexpected lock content hash %q", result.Facts.Lock.ContentHash)
	}
	if result.Facts.Lock.ExpectedContentHash != result.Facts.Lock.ContentHash {
		t.Errorf(
			"expected matching content hashes, got lock %q and expected %q",
			result.Facts.Lock.ContentHash,
			result.Facts.Lock.ExpectedContentHash,
		)
	}
	if result.Facts.Lock.PluginAPIVersion != "2.6.0" {
		t.Errorf("unexpected plugin API version %q", result.Facts.Lock.PluginAPIVersion)
	}

	locked := requirementsWithScopes(
		result.Facts.PlatformRequirements,
		model.RequirementScopeLocked,
		model.RequirementScopeLockedDevelopment,
	)
	expected := []model.Requirement{
		{
			Name:       "ext-intl",
			Kind:       model.RequirementExtension,
			Constraint: "*",
			Scope:      model.RequirementScopeLocked,
			Sources: []model.SourceReference{
				{
					Path:  lockPath,
					Kind:  "composer_lock",
					Field: "/platform/ext-intl",
				},
			},
		},
		{
			Name:       "php",
			Kind:       model.RequirementPHP,
			Constraint: "^8.4",
			Scope:      model.RequirementScopeLocked,
			Sources: []model.SourceReference{
				{
					Path:  lockPath,
					Kind:  "composer_lock",
					Field: "/platform/php",
				},
			},
		},
		{
			Name:       "ext-xdebug",
			Kind:       model.RequirementExtension,
			Constraint: "^3.4",
			Scope:      model.RequirementScopeLockedDevelopment,
			Sources: []model.SourceReference{
				{
					Path:  lockPath,
					Kind:  "composer_lock",
					Field: "/platform-dev/ext-xdebug",
				},
			},
		},
	}
	if len(locked) != len(expected) {
		t.Fatalf("expected %d locked requirements, got %#v", len(expected), locked)
	}
	for index := range expected {
		if !equalRequirement(locked[index], expected[index]) {
			t.Errorf(
				"locked requirement %d\nexpected: %#v\ngot:      %#v",
				index,
				expected[index],
				locked[index],
			)
		}
	}
}

func TestParseProjectCapturesLockedPackageRequirementsAndPlugins(t *testing.T) {
	manifest := readFixture(t, "locked-packages", "composer.json")
	lock := readFixture(t, "locked-packages", "composer.lock")
	lockPath := "/workspace/composer.lock"

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    "/workspace/composer.json",
			Content: manifest,
		},
		Lock: &composer.Document{
			Path:    lockPath,
			Content: lock,
		},
	})
	if err != nil {
		t.Fatalf("parse Composer project: %v", err)
	}

	if len(result.Facts.Lock.Packages) != 3 {
		t.Fatalf("expected three locked packages, got %#v", result.Facts.Lock.Packages)
	}
	runtime := result.Facts.Lock.Packages[1]
	if runtime.Name != "acme/runtime" ||
		runtime.Version != "1.4.0" ||
		runtime.Type != "library" ||
		runtime.Development {
		t.Errorf("unexpected runtime package facts %#v", runtime)
	}
	expectedRuntimeLinks := []model.ComposerLink{
		{
			Name:       "acme/plugin",
			Constraint: "^2.0",
			Source: model.SourceReference{
				Path:  lockPath,
				Kind:  "composer_lock",
				Field: "/packages/1/require/acme~1plugin",
			},
		},
		{
			Name:       "ext-json",
			Constraint: "*",
			Source: model.SourceReference{
				Path:  lockPath,
				Kind:  "composer_lock",
				Field: "/packages/1/require/ext-json",
			},
		},
	}
	assertComposerLinks(t, runtime.Requirements, expectedRuntimeLinks)

	if len(result.Facts.Plugins) != 1 {
		t.Fatalf("expected one Composer plugin, got %#v", result.Facts.Plugins)
	}
	expectedPlugin := model.ComposerPlugin{
		Name:    "acme/plugin",
		Version: "2.1.0",
		Source: model.SourceReference{
			Path:  lockPath,
			Kind:  "composer_lock",
			Field: "/packages/0",
		},
	}
	if result.Facts.Plugins[0] != expectedPlugin {
		t.Errorf("expected plugin %#v, got %#v", expectedPlugin, result.Facts.Plugins[0])
	}

	lockedPackageRequirements := requirementsWithScopes(
		result.Facts.PlatformRequirements,
		model.RequirementScopeLockedPackage,
		model.RequirementScopeLockedPackageConflict,
		model.RequirementScopeLockedDevelopmentPackage,
	)
	expectedNames := []string{
		"composer-plugin-api",
		"php",
		"ext-json",
		"php",
		"composer-runtime-api",
		"ext-xdebug",
	}
	if len(lockedPackageRequirements) != len(expectedNames) {
		t.Fatalf(
			"expected %d locked package requirements, got %#v",
			len(expectedNames),
			lockedPackageRequirements,
		)
	}
	for index, expectedName := range expectedNames {
		if lockedPackageRequirements[index].Name != expectedName {
			t.Errorf(
				"locked package requirement %d expected %q, got %#v",
				index,
				expectedName,
				lockedPackageRequirements[index],
			)
		}
	}
}

func TestParseProjectReportsMalformedLockWithoutLeakingContent(t *testing.T) {
	manifest := readFixture(t, "malformed-lock", "composer.json")
	lock := readFixture(t, "malformed-lock", "composer.lock")

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    "/workspace/composer.json",
			Content: manifest,
		},
		Lock: &composer.Document{
			Path:    "/workspace/composer.lock",
			Content: lock,
		},
	})
	if err != nil {
		t.Fatalf("expected manifest facts with a lock diagnostic, got %v", err)
	}

	if result.Facts.Lock.Status != model.ComposerLockInvalid {
		t.Errorf("expected invalid lock status, got %q", result.Facts.Lock.Status)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one malformed lock diagnostic, got %#v", result.Diagnostics)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Code != "ELEFANTE_COMPOSER_LOCK_MALFORMED" {
		t.Errorf("unexpected diagnostic code %q", diagnostic.Code)
	}
	if diagnostic.Severity != model.SeverityError {
		t.Errorf("expected error severity, got %q", diagnostic.Severity)
	}

	encoded, err := json.Marshal(diagnostic)
	if err != nil {
		t.Fatalf("marshal malformed lock diagnostic: %v", err)
	}
	if strings.Contains(string(encoded), "synthetic-secret-value") {
		t.Fatalf("malformed lock diagnostic leaked input content: %s", encoded)
	}
}

func TestParseProjectReportsInternallyInconsistentLock(t *testing.T) {
	manifest := readFixture(t, "inconsistent-lock", "composer.json")
	lock := readFixture(t, "inconsistent-lock", "composer.lock")

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    "/workspace/composer.json",
			Content: manifest,
		},
		Lock: &composer.Document{
			Path:    "/workspace/composer.lock",
			Content: lock,
		},
	})
	if err != nil {
		t.Fatalf("expected manifest facts with a lock diagnostic, got %v", err)
	}

	if result.Facts.Lock.Status != model.ComposerLockInconsistent {
		t.Errorf("expected inconsistent lock status, got %q", result.Facts.Lock.Status)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one inconsistent lock diagnostic, got %#v", result.Diagnostics)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Code != "ELEFANTE_COMPOSER_LOCK_INCONSISTENT" {
		t.Errorf("unexpected diagnostic code %q", diagnostic.Code)
	}
	if diagnostic.Severity != model.SeverityError {
		t.Errorf("expected error severity, got %q", diagnostic.Severity)
	}
	if len(diagnostic.Sources) != 2 {
		t.Fatalf("expected both duplicate package sources, got %#v", diagnostic.Sources)
	}
}

func TestParseProjectReportsStaleLockFile(t *testing.T) {
	manifest := readFixture(t, "stale-lock", "composer.json")
	lock := readFixture(t, "stale-lock", "composer.lock")

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    "/workspace/composer.json",
			Content: manifest,
		},
		Lock: &composer.Document{
			Path:    "/workspace/composer.lock",
			Content: lock,
		},
	})
	if err != nil {
		t.Fatalf("parse Composer project: %v", err)
	}

	if result.Facts.Lock.Status != model.ComposerLockStale {
		t.Errorf("expected stale lock status, got %q", result.Facts.Lock.Status)
	}
	if result.Facts.Lock.ExpectedContentHash == result.Facts.Lock.ContentHash {
		t.Errorf("expected stale content hashes to differ")
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one stale lock diagnostic, got %#v", result.Diagnostics)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Code != "ELEFANTE_COMPOSER_LOCK_STALE" {
		t.Errorf("unexpected diagnostic code %q", diagnostic.Code)
	}
	if diagnostic.Severity != model.SeverityWarning {
		t.Errorf("expected warning severity, got %q", diagnostic.Severity)
	}
}

func TestParseProjectRejectsNullRequirementObject(t *testing.T) {
	_, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    "/workspace/composer.json",
			Content: []byte(`{"name":"acme/example","require":null}`),
		},
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed requirements error, got %v", err)
	}
	if commandError.Code != model.ErrorRequirements {
		t.Errorf("expected requirements code, got %q", commandError.Code)
	}
	if len(commandError.Sources) != 1 {
		t.Fatalf("expected one schema source, got %#v", commandError.Sources)
	}
	if commandError.Sources[0].Field != "/require" {
		t.Errorf("expected /require source field, got %q", commandError.Sources[0].Field)
	}
}

func TestParseProjectReportsNullLockPackagesAsMalformed(t *testing.T) {
	manifest := readFixture(t, "locked-platform", "composer.json")
	lock := []byte(`{
		"content-hash": "8c785f5fae6d3a41bcb172e2b2bcd347",
		"packages": null,
		"packages-dev": [],
		"platform": {},
		"platform-dev": {}
	}`)

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    "/workspace/composer.json",
			Content: manifest,
		},
		Lock: &composer.Document{
			Path:    "/workspace/composer.lock",
			Content: lock,
		},
	})
	if err != nil {
		t.Fatalf("expected manifest facts with lock diagnostic, got %v", err)
	}
	if result.Facts.Lock.Status != model.ComposerLockInvalid {
		t.Errorf("expected invalid lock status, got %q", result.Facts.Lock.Status)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one malformed lock diagnostic, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "ELEFANTE_COMPOSER_LOCK_MALFORMED" {
		t.Errorf("unexpected diagnostic code %q", result.Diagnostics[0].Code)
	}
	if result.Diagnostics[0].Sources[0].Field != "/packages" {
		t.Errorf("expected /packages source, got %#v", result.Diagnostics[0].Sources)
	}
}

func TestParseProjectRejectsUnpairedManifestSurrogateWithoutLeakingContent(t *testing.T) {
	_, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path: "/workspace/composer.json",
			Content: []byte(
				`{"name":"acme/\ud800synthetic-surrogate-secret"}`,
			),
		},
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed requirements error, got %v", err)
	}
	encoded, marshalErr := json.Marshal(commandError)
	if marshalErr != nil {
		t.Fatalf("marshal requirements error: %v", marshalErr)
	}
	if strings.Contains(string(encoded), "synthetic-surrogate-secret") {
		t.Fatalf("requirements error leaked malformed content: %s", encoded)
	}
}

func TestParseProjectReportsUnpairedLockSurrogateAsMalformed(t *testing.T) {
	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    "/workspace/composer.json",
			Content: []byte(`{"name":"acme/example"}`),
		},
		Lock: &composer.Document{
			Path: "/workspace/composer.lock",
			Content: []byte(`{
				"content-hash": "deadbeefdeadbeefdeadbeefdeadbeef",
				"packages": [
					{
						"name": "acme/\ud800synthetic-surrogate-secret",
						"version": "1.0.0"
					}
				]
			}`),
		},
	})
	if err != nil {
		t.Fatalf("expected a lock diagnostic, got %v", err)
	}
	if result.Facts.Lock.Status != model.ComposerLockInvalid {
		t.Errorf("expected invalid lock status, got %q", result.Facts.Lock.Status)
	}
	encoded, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		t.Fatalf("marshal parser result: %v", marshalErr)
	}
	if strings.Contains(string(encoded), "synthetic-surrogate-secret") {
		t.Fatalf("lock diagnostic leaked malformed content: %s", encoded)
	}
}

func TestParseProjectAcceptsPairedManifestSurrogate(t *testing.T) {
	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    "/workspace/composer.json",
			Content: []byte(`{"name":"acme/\ud83d\udc18"}`),
		},
	})
	if err != nil {
		t.Fatalf("parse paired surrogate: %v", err)
	}
	if result.Facts.Manifest.Name != "acme/🐘" {
		t.Errorf("expected decoded elephant name, got %q", result.Facts.Manifest.Name)
	}
}

func requirementsWithScopes(
	requirements []model.Requirement,
	scopes ...model.RequirementScope,
) []model.Requirement {
	allowed := make(map[model.RequirementScope]struct{}, len(scopes))
	for _, scope := range scopes {
		allowed[scope] = struct{}{}
	}

	var result []model.Requirement
	for _, requirement := range requirements {
		if _, exists := allowed[requirement.Scope]; exists {
			result = append(result, requirement)
		}
	}

	return result
}

func scriptDigest(t *testing.T, commands []string) string {
	t.Helper()

	encoded, err := json.Marshal(commands)
	if err != nil {
		t.Fatalf("encode script commands: %v", err)
	}
	sum := sha256.Sum256(encoded)

	return "sha256:" + hex.EncodeToString(sum[:])
}

func readFixture(t *testing.T, parts ...string) []byte {
	t.Helper()

	pathParts := append([]string{"..", "..", "testdata", "fixtures", "composer"}, parts...)
	content, err := os.ReadFile(filepath.Join(pathParts...))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	return content
}

func assertComposerLinks(t *testing.T, got []model.ComposerLink, expected []model.ComposerLink) {
	t.Helper()

	if len(got) != len(expected) {
		t.Fatalf("expected %d Composer links, got %#v", len(expected), got)
	}
	for index := range expected {
		if got[index] != expected[index] {
			t.Errorf("Composer link %d\nexpected: %#v\ngot:      %#v", index, expected[index], got[index])
		}
	}
}

func equalRequirement(got model.Requirement, expected model.Requirement) bool {
	if got.Name != expected.Name ||
		got.Kind != expected.Kind ||
		got.Constraint != expected.Constraint ||
		got.Scope != expected.Scope ||
		got.Optional != expected.Optional ||
		len(got.Sources) != len(expected.Sources) {
		return false
	}

	for index := range expected.Sources {
		if got.Sources[index] != expected.Sources[index] {
			return false
		}
	}

	return true
}
