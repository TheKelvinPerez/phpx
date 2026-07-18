package frameworks_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elefantephp/elefante/internal/composer"
	"github.com/elefantephp/elefante/internal/frameworks"
	"github.com/elefantephp/elefante/internal/model"
)

func TestDetectsLaravelApplicationFromComposerAndFileEvidence(t *testing.T) {
	root := frameworkFixture(t, "laravel-application")
	composerFacts := parseFixtureComposer(t, root)

	result, err := frameworks.Detect(frameworks.Request{
		ComposerRoot: root,
		Composer:     composerFacts,
	})
	if err != nil {
		t.Fatalf("detect frameworks: %v", err)
	}

	laravel := frameworkFact(t, result.Facts, model.FrameworkLaravelApplication)
	if laravel.Confidence != model.FrameworkConfidenceHigh {
		t.Fatalf("expected high confidence, got %q", laravel.Confidence)
	}
	if !laravel.Primary {
		t.Fatal("expected Laravel application to be the primary adapter")
	}
	if len(laravel.Evidence) != 4 {
		t.Fatalf("expected four Laravel evidence records, got %#v", laravel.Evidence)
	}

	generic := frameworkFact(t, result.Facts, model.FrameworkGenericComposer)
	if generic.Primary {
		t.Fatal("expected generic Composer to remain a nonprimary fallback")
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no framework diagnostics, got %#v", result.Diagnostics)
	}
}

func TestDetectsLaravelPackageWithoutApplicationMarkers(t *testing.T) {
	root := frameworkFixture(t, "laravel-package")
	composerFacts := parseFixtureComposer(t, root)

	result, err := frameworks.Detect(frameworks.Request{
		ComposerRoot: root,
		Composer:     composerFacts,
	})
	if err != nil {
		t.Fatalf("detect frameworks: %v", err)
	}

	laravel := frameworkFact(t, result.Facts, model.FrameworkLaravelPackage)
	if laravel.Confidence != model.FrameworkConfidenceHigh {
		t.Fatalf("expected high confidence, got %q", laravel.Confidence)
	}
	if !laravel.Primary {
		t.Fatal("expected Laravel package to be the primary adapter")
	}
	if len(laravel.Evidence) != 2 {
		t.Fatalf("expected package requirement and type evidence, got %#v", laravel.Evidence)
	}
	if hasFrameworkFact(result.Facts, model.FrameworkLaravelApplication) {
		t.Fatal("expected no Laravel application fact without bootstrap markers")
	}
}

func TestDetectsBedrockWordPressWithoutAssumingRootWordPressCore(t *testing.T) {
	root := frameworkFixture(t, "bedrock-wordpress")
	composerFacts := parseFixtureComposer(t, root)

	result, err := frameworks.Detect(frameworks.Request{
		ComposerRoot: root,
		Composer:     composerFacts,
	})
	if err != nil {
		t.Fatalf("detect frameworks: %v", err)
	}

	bedrock := frameworkFact(t, result.Facts, model.FrameworkBedrockWordPress)
	if bedrock.Confidence != model.FrameworkConfidenceHigh {
		t.Fatalf("expected high confidence, got %q", bedrock.Confidence)
	}
	if !bedrock.Primary {
		t.Fatal("expected Bedrock WordPress to be the primary adapter")
	}
	if len(bedrock.Evidence) != 4 {
		t.Fatalf("expected four Bedrock evidence records, got %#v", bedrock.Evidence)
	}
	for _, evidence := range bedrock.Evidence {
		if evidence.Source.Path == filepath.Join(root, "wp-load.php") {
			t.Fatal("expected no repository root WordPress core assumption")
		}
	}
}

func TestDetectsSymfonyApplicationFromComposerAndConsoleMarkers(t *testing.T) {
	root := frameworkFixture(t, "symfony")
	composerFacts := parseFixtureComposer(t, root)

	result, err := frameworks.Detect(frameworks.Request{
		ComposerRoot: root,
		Composer:     composerFacts,
	})
	if err != nil {
		t.Fatalf("detect frameworks: %v", err)
	}

	symfony := frameworkFact(t, result.Facts, model.FrameworkSymfonyApplication)
	if symfony.Confidence != model.FrameworkConfidenceHigh {
		t.Fatalf("expected high confidence, got %q", symfony.Confidence)
	}
	if !symfony.Primary {
		t.Fatal("expected Symfony application to be the primary adapter")
	}
	if len(symfony.Evidence) != 4 {
		t.Fatalf("expected four Symfony evidence records, got %#v", symfony.Evidence)
	}
}

func TestKeepsGenericComposerAsPrimaryFallback(t *testing.T) {
	root := frameworkFixture(t, "generic-composer")
	composerFacts := parseFixtureComposer(t, root)

	result, err := frameworks.Detect(frameworks.Request{
		ComposerRoot: root,
		Composer:     composerFacts,
	})
	if err != nil {
		t.Fatalf("detect frameworks: %v", err)
	}

	if len(result.Facts) != 1 {
		t.Fatalf("expected only generic Composer, got %#v", result.Facts)
	}
	generic := frameworkFact(t, result.Facts, model.FrameworkGenericComposer)
	if generic.Confidence != model.FrameworkConfidenceFallback {
		t.Fatalf("expected fallback confidence, got %q", generic.Confidence)
	}
	if !generic.Primary {
		t.Fatal("expected generic Composer to be primary without framework evidence")
	}
}

func TestReportsConflictingStrongFrameworkEvidence(t *testing.T) {
	root := frameworkFixture(t, "conflicting")
	composerFacts := parseFixtureComposer(t, root)

	result, err := frameworks.Detect(frameworks.Request{
		ComposerRoot: root,
		Composer:     composerFacts,
	})
	if err != nil {
		t.Fatalf("detect frameworks: %v", err)
	}

	frameworkFact(t, result.Facts, model.FrameworkLaravelApplication)
	frameworkFact(t, result.Facts, model.FrameworkSymfonyApplication)
	for _, fact := range result.Facts {
		if fact.Primary {
			t.Fatalf("expected no arbitrary primary adapter, got %#v", fact)
		}
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one conflict diagnostic, got %#v", result.Diagnostics)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Code != "ELEFANTE_FRAMEWORK_CONFLICT" {
		t.Fatalf("expected framework conflict code, got %q", diagnostic.Code)
	}
	if diagnostic.Severity != model.SeverityError {
		t.Fatalf("expected error severity, got %q", diagnostic.Severity)
	}
	if len(diagnostic.Sources) < 2 {
		t.Fatalf("expected evidence sources for both frameworks, got %#v", diagnostic.Sources)
	}
}

func TestRejectsFrameworkMarkerThroughEscapingParentSymlink(t *testing.T) {
	root := t.TempDir()
	outsideRoot := t.TempDir()
	composerPath := filepath.Join(root, "composer.json")
	composerContent := []byte(`{
		"name": "acme/laravel-application",
		"type": "project",
		"require": {
			"laravel/framework": "^13.0"
		}
	}`)
	if err := os.WriteFile(composerPath, composerContent, 0o644); err != nil {
		t.Fatalf("write Composer metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "artisan"), []byte("inert\n"), 0o644); err != nil {
		t.Fatalf("write Artisan marker: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "public"), 0o755); err != nil {
		t.Fatalf("create public directory: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "public", "index.php"),
		[]byte("inert\n"),
		0o644,
	); err != nil {
		t.Fatalf("write front controller marker: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(outsideRoot, "app.php"),
		[]byte("inert\n"),
		0o644,
	); err != nil {
		t.Fatalf("write outside bootstrap marker: %v", err)
	}
	if err := os.Symlink(outsideRoot, filepath.Join(root, "bootstrap")); err != nil {
		t.Fatalf("create escaping bootstrap symlink: %v", err)
	}

	composerResult, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    composerPath,
			Content: composerContent,
		},
	})
	if err != nil {
		t.Fatalf("parse Composer metadata: %v", err)
	}
	_, err = frameworks.Detect(frameworks.Request{
		ComposerRoot: root,
		Composer:     composerResult.Facts,
	})

	var commandError *model.Error
	if !errors.As(err, &commandError) {
		t.Fatalf("expected typed discovery error, got %v", err)
	}
	cause := errors.Unwrap(commandError)
	if cause == nil || !strings.Contains(cause.Error(), "outside the Composer root") {
		t.Fatalf("expected framework boundary error, got %#v", commandError)
	}
}

func frameworkFixture(t *testing.T, name string) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join(
		"..",
		"..",
		"testdata",
		"fixtures",
		"frameworks",
		name,
	))
	if err != nil {
		t.Fatalf("resolve framework fixture: %v", err)
	}

	return root
}

func parseFixtureComposer(t *testing.T, root string) model.ComposerFacts {
	t.Helper()

	path := filepath.Join(root, "composer.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture Composer metadata: %v", err)
	}
	result, err := composer.ParseProject(composer.ProjectInput{
		Manifest: composer.Document{
			Path:    path,
			Content: content,
		},
	})
	if err != nil {
		t.Fatalf("parse fixture Composer metadata: %v", err)
	}

	return result.Facts
}

func frameworkFact(
	t *testing.T,
	facts []model.FrameworkFact,
	kind model.FrameworkKind,
) model.FrameworkFact {
	t.Helper()

	for _, fact := range facts {
		if fact.Kind == kind {
			return fact
		}
	}
	t.Fatalf("expected framework fact %q, got %#v", kind, facts)

	return model.FrameworkFact{}
}

func hasFrameworkFact(
	facts []model.FrameworkFact,
	kind model.FrameworkKind,
) bool {
	for _, fact := range facts {
		if fact.Kind == kind {
			return true
		}
	}

	return false
}
