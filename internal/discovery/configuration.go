package discovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elefantephp/elefante/internal/config"
	"github.com/elefantephp/elefante/internal/model"
)

type loadedConfiguration struct {
	metadata metadataFile
	result   config.Result
}

type configurationSelection struct {
	facts        model.ConfigFacts
	diagnostics  []model.Diagnostic
	fingerprints []model.InputFingerprint
}

func (selection *configurationSelection) validateComposerRoot(
	composerRoot string,
) {
	if selection.facts.Path == "" ||
		len(selection.diagnostics) != 0 ||
		selection.facts.Project.ComposerRoot == composerRoot {
		return
	}

	selection.diagnostics = append(selection.diagnostics, model.Diagnostic{
		Code:     "ELEFANTE_CONFIG_PROJECT_MISMATCH",
		Severity: model.SeverityError,
		Message:  "Elefante configuration selects a different Composer root.",
		Detail: fmt.Sprintf(
			"Configuration selects %s, but discovery selected %s.",
			selection.facts.Project.ComposerRoot,
			composerRoot,
		),
		Hint: "Pass --project for the configured Composer root or correct project.composer_root.",
		Sources: []model.SourceReference{
			{
				Path:  selection.facts.Path,
				Kind:  "elefante_config",
				Field: "/project/composer_root",
			},
			{
				Path: filepath.Join(composerRoot, "composer.json"),
				Kind: "composer_manifest",
			},
		},
	})
}

func loadConfiguration(
	path string,
	repositoryRoot string,
	maxSize int64,
) (loadedConfiguration, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return loadedConfiguration{}, model.WrapError(
			model.ErrorDiscovery,
			"Could not normalize the Elefante configuration path.",
			err,
		)
	}
	absolutePath = filepath.Clean(absolutePath)

	metadata, err := readMetadataFile(
		absolutePath,
		repositoryRoot,
		maxSize,
		"elefante_config",
		"Elefante configuration",
	)
	if err != nil {
		return loadedConfiguration{}, err
	}
	resolvedPath, err := filepath.EvalSymlinks(absolutePath)
	if err != nil {
		return loadedConfiguration{}, metadataError(
			absolutePath,
			"elefante_config",
			"Could not resolve Elefante configuration.",
			err,
		)
	}
	metadata.fingerprint.Path = resolvedPath

	return loadedConfiguration{
		metadata: metadata,
		result: config.Parse(config.Document{
			Path:           resolvedPath,
			RepositoryRoot: repositoryRoot,
			Content:        metadata.content,
		}),
	}, nil
}

func loadOptionalConfiguration(
	path string,
	repositoryRoot string,
	maxSize int64,
) (*loadedConfiguration, error) {
	_, err := os.Lstat(path)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return nil, nil
	default:
		return nil, metadataError(
			path,
			"elefante_config",
			"Could not inspect Elefante configuration.",
			err,
		)
	}

	loaded, err := loadConfiguration(path, repositoryRoot, maxSize)
	if err != nil {
		return nil, err
	}

	return &loaded, nil
}

func selectConfiguration(
	explicitPath string,
	repositoryRoot string,
	composerRoot string,
	maxSize int64,
) (configurationSelection, error) {
	if explicitPath != "" {
		loaded, err := loadConfiguration(explicitPath, repositoryRoot, maxSize)
		if err != nil {
			return configurationSelection{}, err
		}

		return selectionFromConfigurations(&loaded), nil
	}

	repositoryPath := filepath.Join(repositoryRoot, "elefante.toml")
	projectPath := filepath.Join(composerRoot, "elefante.toml")
	repositoryConfig, err := loadOptionalConfiguration(
		repositoryPath,
		repositoryRoot,
		maxSize,
	)
	if err != nil {
		return configurationSelection{}, err
	}
	if repositoryPath == projectPath {
		return selectionFromConfigurations(repositoryConfig), nil
	}

	projectConfig, err := loadOptionalConfiguration(
		projectPath,
		repositoryRoot,
		maxSize,
	)
	if err != nil {
		return configurationSelection{}, err
	}

	switch {
	case repositoryConfig == nil:
		return selectionFromConfigurations(projectConfig), nil
	case projectConfig == nil:
		return selectionFromConfigurations(repositoryConfig), nil
	case configurationSelectsRoot(repositoryConfig, composerRoot):
		selection := selectionFromConfigurations(projectConfig)
		selection.fingerprints = append(
			[]model.InputFingerprint{repositoryConfig.metadata.fingerprint},
			selection.fingerprints...,
		)

		return selection, nil
	default:
		return configurationSelection{
			diagnostics: []model.Diagnostic{
				{
					Code:     "ELEFANTE_CONFIG_AMBIGUOUS",
					Severity: model.SeverityError,
					Message:  "Repository and project Elefante configurations are ambiguous.",
					Detail: fmt.Sprintf(
						"%s does not select Composer root %s.",
						repositoryPath,
						composerRoot,
					),
					Hint: "Set project.composer_root in the repository configuration or pass --config.",
					Sources: []model.SourceReference{
						{Path: repositoryPath, Kind: "elefante_config"},
						{Path: projectPath, Kind: "elefante_config"},
					},
				},
			},
			fingerprints: []model.InputFingerprint{
				repositoryConfig.metadata.fingerprint,
				projectConfig.metadata.fingerprint,
			},
		}, nil
	}
}

func selectionFromConfigurations(
	configuration *loadedConfiguration,
) configurationSelection {
	if configuration == nil {
		return configurationSelection{}
	}

	return configurationSelection{
		facts:        configuration.result.Facts,
		diagnostics:  configuration.result.Diagnostics,
		fingerprints: []model.InputFingerprint{configuration.metadata.fingerprint},
	}
}

func configurationSelectsRoot(
	configuration *loadedConfiguration,
	composerRoot string,
) bool {
	return configuration != nil &&
		len(configuration.result.Diagnostics) == 0 &&
		configuration.result.Facts.Project.ComposerRoot == composerRoot
}

func configuredComposerRoot(
	explicitPath string,
	repositoryRoot string,
	candidates []string,
	maxSize int64,
) (string, bool, error) {
	var configuration *loadedConfiguration
	var err error
	if explicitPath != "" {
		loaded, loadErr := loadConfiguration(
			explicitPath,
			repositoryRoot,
			maxSize,
		)
		if loadErr != nil {
			return "", false, loadErr
		}
		configuration = &loaded
	} else {
		configuration, err = loadOptionalConfiguration(
			filepath.Join(repositoryRoot, "elefante.toml"),
			repositoryRoot,
			maxSize,
		)
		if err != nil {
			return "", false, err
		}
	}
	if configuration == nil || len(configuration.result.Diagnostics) != 0 {
		return "", false, nil
	}

	selectedRoot := configuration.result.Facts.Project.ComposerRoot
	for _, candidate := range candidates {
		if selectedRoot == candidate {
			return selectedRoot, true, nil
		}
	}

	return "", false, nil
}
