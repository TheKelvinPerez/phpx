package frameworks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elefantephp/elefante/internal/model"
	projectpaths "github.com/elefantephp/elefante/internal/paths"
)

type Request struct {
	ComposerRoot string
	Composer     model.ComposerFacts
}

type Result struct {
	Facts       []model.FrameworkFact
	Diagnostics []model.Diagnostic
}

func Detect(request Request) (Result, error) {
	generic := model.FrameworkFact{
		Kind:       model.FrameworkGenericComposer,
		Confidence: model.FrameworkConfidenceFallback,
		Primary:    true,
		Evidence: []model.FrameworkEvidence{
			{
				Kind:        "composer_manifest",
				Description: "The selected project has valid Composer metadata.",
				Source: model.SourceReference{
					Path: request.Composer.Manifest.Path,
					Kind: "composer_manifest",
				},
			},
		},
	}
	result := Result{Facts: []model.FrameworkFact{generic}}

	laravel, detected, err := detectLaravel(request)
	if err != nil {
		return Result{}, err
	}
	if detected {
		result.Facts = append(result.Facts, laravel)
	}

	bedrock, detected, err := detectBedrock(request)
	if err != nil {
		return Result{}, err
	}
	if detected {
		result.Facts = append(result.Facts, bedrock)
	}

	symfony, detected, err := detectSymfony(request)
	if err != nil {
		return Result{}, err
	}
	if detected {
		result.Facts = append(result.Facts, symfony)
	}

	if len(result.Facts) == 2 {
		result.Facts[0].Primary = false
		result.Facts[1].Primary = true
	} else if len(result.Facts) > 2 {
		result.Facts[0].Primary = false
		sources := make([]model.SourceReference, 0, len(result.Facts)-1)
		for _, fact := range result.Facts[1:] {
			if len(fact.Evidence) > 0 {
				sources = append(sources, fact.Evidence[0].Source)
			}
		}
		result.Diagnostics = append(result.Diagnostics, model.Diagnostic{
			Code:     "ELEFANTE_FRAMEWORK_CONFLICT",
			Severity: model.SeverityError,
			Message:  "The project contains conflicting strong framework evidence.",
			Detail:   "Elefante will not select a primary framework adapter while several application frameworks have high confidence.",
			Hint:     "Remove stale framework requirements or markers before selecting an adapter.",
			Sources:  sources,
		})
	}

	return result, nil
}

func detectLaravel(request Request) (model.FrameworkFact, bool, error) {
	laravelRequirement, hasLaravelFramework := composerRequirement(
		request.Composer.Manifest.Requirements,
		"laravel/framework",
	)
	laravelPackageRequirement, hasLaravelPackage := laravelEcosystemRequirement(
		request.Composer.Manifest.Requirements,
		request.Composer.Manifest.DevelopmentRequirements,
	)
	if request.Composer.Manifest.Type == "library" && hasLaravelPackage {
		return model.FrameworkFact{
			Kind:       model.FrameworkLaravelPackage,
			Confidence: model.FrameworkConfidenceHigh,
			Evidence: []model.FrameworkEvidence{
				{
					Kind:        "composer_requirement",
					Description: "The package requires Laravel or Illuminate components.",
					Source:      laravelPackageRequirement.Source,
				},
				{
					Kind:        "composer_type",
					Description: "Composer identifies the root package as a library.",
					Source: model.SourceReference{
						Path:  request.Composer.Manifest.Path,
						Kind:  "composer_manifest",
						Field: "/type",
					},
				},
			},
		}, true, nil
	}
	if !hasLaravelFramework {
		return model.FrameworkFact{}, false, nil
	}

	evidence := []model.FrameworkEvidence{
		{
			Kind:        "composer_requirement",
			Description: "The root project requires laravel/framework.",
			Source:      laravelRequirement.Source,
		},
	}
	for _, marker := range []struct {
		path        string
		description string
	}{
		{path: "artisan", description: "The project contains an Artisan entry point."},
		{path: filepath.Join("bootstrap", "app.php"), description: "The project contains Laravel bootstrap metadata."},
		{path: filepath.Join("public", "index.php"), description: "The project contains a conventional public front controller."},
	} {
		present, err := regularMarker(request.ComposerRoot, marker.path)
		if err != nil {
			return model.FrameworkFact{}, false, err
		}
		if present {
			evidence = append(evidence, model.FrameworkEvidence{
				Kind:        "file_marker",
				Description: marker.description,
				Source: model.SourceReference{
					Path: filepath.Join(request.ComposerRoot, marker.path),
					Kind: "framework_marker",
				},
			})
		}
	}
	if len(evidence) != 4 {
		return model.FrameworkFact{}, false, nil
	}

	return model.FrameworkFact{
		Kind:       model.FrameworkLaravelApplication,
		Confidence: model.FrameworkConfidenceHigh,
		Evidence:   evidence,
	}, true, nil
}

func detectBedrock(request Request) (model.FrameworkFact, bool, error) {
	wordPressRequirement, found := composerRequirement(
		request.Composer.Manifest.Requirements,
		"roots/wordpress",
	)
	if !found {
		return model.FrameworkFact{}, false, nil
	}

	evidence := []model.FrameworkEvidence{
		{
			Kind:        "composer_requirement",
			Description: "The root project requires roots/wordpress.",
			Source:      wordPressRequirement.Source,
		},
	}
	fileMarkers := []struct {
		path        string
		description string
	}{
		{
			path:        filepath.Join("config", "application.php"),
			description: "The project contains Bedrock application configuration.",
		},
		{
			path:        filepath.Join("web", "wp-config.php"),
			description: "The project contains Bedrock WordPress configuration.",
		},
	}
	for _, marker := range fileMarkers {
		present, err := regularMarker(request.ComposerRoot, marker.path)
		if err != nil {
			return model.FrameworkFact{}, false, err
		}
		if present {
			evidence = append(evidence, model.FrameworkEvidence{
				Kind:        "file_marker",
				Description: marker.description,
				Source: model.SourceReference{
					Path: filepath.Join(request.ComposerRoot, marker.path),
					Kind: "framework_marker",
				},
			})
		}
	}
	appDirectory := filepath.Join("web", "app")
	present, err := directoryMarker(request.ComposerRoot, appDirectory)
	if err != nil {
		return model.FrameworkFact{}, false, err
	}
	if present {
		evidence = append(evidence, model.FrameworkEvidence{
			Kind:        "directory_marker",
			Description: "The project uses Bedrock's web/app content directory.",
			Source: model.SourceReference{
				Path: filepath.Join(request.ComposerRoot, appDirectory),
				Kind: "framework_marker",
			},
		})
	}
	if len(evidence) != 4 {
		return model.FrameworkFact{}, false, nil
	}

	return model.FrameworkFact{
		Kind:       model.FrameworkBedrockWordPress,
		Confidence: model.FrameworkConfidenceHigh,
		Evidence:   evidence,
	}, true, nil
}

func detectSymfony(request Request) (model.FrameworkFact, bool, error) {
	frameworkRequirement, found := composerRequirement(
		request.Composer.Manifest.Requirements,
		"symfony/framework-bundle",
	)
	if !found {
		return model.FrameworkFact{}, false, nil
	}

	evidence := []model.FrameworkEvidence{
		{
			Kind:        "composer_requirement",
			Description: "The root project requires symfony/framework-bundle.",
			Source:      frameworkRequirement.Source,
		},
	}
	for _, marker := range []struct {
		path        string
		description string
	}{
		{
			path:        filepath.Join("bin", "console"),
			description: "The project contains a Symfony Console entry point.",
		},
		{
			path:        filepath.Join("config", "bundles.php"),
			description: "The project contains Symfony bundle configuration.",
		},
		{
			path:        filepath.Join("public", "index.php"),
			description: "The project contains a conventional public front controller.",
		},
	} {
		present, err := regularMarker(request.ComposerRoot, marker.path)
		if err != nil {
			return model.FrameworkFact{}, false, err
		}
		if present {
			evidence = append(evidence, model.FrameworkEvidence{
				Kind:        "file_marker",
				Description: marker.description,
				Source: model.SourceReference{
					Path: filepath.Join(request.ComposerRoot, marker.path),
					Kind: "framework_marker",
				},
			})
		}
	}
	if len(evidence) != 4 {
		return model.FrameworkFact{}, false, nil
	}

	return model.FrameworkFact{
		Kind:       model.FrameworkSymfonyApplication,
		Confidence: model.FrameworkConfidenceHigh,
		Evidence:   evidence,
	}, true, nil
}

func composerRequirement(
	requirements []model.ComposerLink,
	name string,
) (model.ComposerLink, bool) {
	for _, requirement := range requirements {
		if requirement.Name == name {
			return requirement, true
		}
	}

	return model.ComposerLink{}, false
}

func laravelEcosystemRequirement(
	requirements ...[]model.ComposerLink,
) (model.ComposerLink, bool) {
	for _, links := range requirements {
		for _, requirement := range links {
			if requirement.Name == "laravel/framework" ||
				strings.HasPrefix(requirement.Name, "illuminate/") {
				return requirement, true
			}
		}
	}

	return model.ComposerLink{}, false
}

func regularMarker(root string, relativePath string) (bool, error) {
	return marker(root, relativePath, false)
}

func directoryMarker(root string, relativePath string) (bool, error) {
	return marker(root, relativePath, true)
}

func marker(root string, relativePath string, wantDirectory bool) (bool, error) {
	path := filepath.Join(root, relativePath)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, frameworkInspectionError(path, err)
	}

	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false, frameworkInspectionError(path, err)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false, frameworkInspectionError(path, err)
	}
	if !projectpaths.Contains(resolvedRoot, resolvedPath) {
		return false, frameworkInspectionError(
			path,
			fmt.Errorf("marker resolves outside the Composer root"),
		)
	}
	info, err = os.Stat(resolvedPath)
	if err != nil {
		return false, frameworkInspectionError(path, err)
	}

	if wantDirectory {
		return info.IsDir(), nil
	}

	return info.Mode().IsRegular(), nil
}

func frameworkInspectionError(path string, cause error) *model.Error {
	commandError := model.WrapError(
		model.ErrorDiscovery,
		"Could not inspect framework evidence.",
		cause,
	)
	commandError.Sources = []model.SourceReference{
		{
			Path: path,
			Kind: "framework_marker",
		},
	}

	return commandError
}
