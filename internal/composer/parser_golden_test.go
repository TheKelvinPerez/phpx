package composer_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/model"
)

func TestParseProjectMatchesSourceAttributionGolden(t *testing.T) {
	result := parseFixtureProject(t, "locked-packages")
	if len(result.Facts.Manifest.Requirements) != 1 ||
		len(result.Facts.Manifest.DevelopmentRequirements) != 1 ||
		len(result.Facts.Lock.Packages) != 3 ||
		len(result.Facts.Plugins) != 1 {
		t.Fatalf("fixture no longer has the expected golden projection shape")
	}

	golden := struct {
		RootRequirement        model.ComposerLink    `json:"root_requirement"`
		DevelopmentRequirement model.ComposerLink    `json:"development_requirement"`
		RuntimePackage         model.ComposerPackage `json:"runtime_package"`
		Plugin                 model.ComposerPlugin  `json:"plugin"`
		PlatformRequirements   []model.Requirement   `json:"platform_requirements"`
	}{
		RootRequirement:        result.Facts.Manifest.Requirements[0],
		DevelopmentRequirement: result.Facts.Manifest.DevelopmentRequirements[0],
		RuntimePackage:         result.Facts.Lock.Packages[1],
		Plugin:                 result.Facts.Plugins[0],
		PlatformRequirements:   result.Facts.PlatformRequirements,
	}

	assertComposerGolden(t, "source-attribution.json", golden)
}

func TestParseProjectMatchesRedactedDiagnosticGolden(t *testing.T) {
	result := parseFixtureProject(t, "malformed-lock")
	golden := struct {
		Status      model.ComposerLockStatus `json:"status"`
		Diagnostics []model.Diagnostic       `json:"diagnostics"`
	}{
		Status:      result.Facts.Lock.Status,
		Diagnostics: result.Diagnostics,
	}

	encoded := assertComposerGolden(t, "redacted-diagnostic.json", golden)
	if bytes.Contains(encoded, []byte("synthetic-secret-value")) {
		t.Fatalf("redacted diagnostic golden contains malformed lock content")
	}
}

func parseFixtureProject(t *testing.T, fixture string) composer.ParseResult {
	t.Helper()

	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    "/workspace/composer.json",
			Content: readFixture(t, fixture, "composer.json"),
		},
		Lock: &composer.Document{
			Path:    "/workspace/composer.lock",
			Content: readFixture(t, fixture, "composer.lock"),
		},
	})
	if err != nil {
		t.Fatalf("parse Composer fixture %q: %v", fixture, err)
	}

	return result
}

func assertComposerGolden(t *testing.T, name string, value any) []byte {
	t.Helper()

	actual, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal Composer golden %q: %v", name, err)
	}
	actual = append(actual, '\n')

	path := filepath.Join(
		"..",
		"..",
		"testdata",
		"golden",
		"composer",
		name,
	)
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Composer golden %s: %v\nactual:\n%s", path, err, actual)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatalf(
			"Composer golden %q changed\nexpected:\n%s\nactual:\n%s",
			name,
			expected,
			actual,
		)
	}

	return actual
}
